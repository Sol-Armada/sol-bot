package onboarding

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
)

var choiceMade map[string]string = map[string]string{}

var basicQuestions *discordgo.MessageSend = &discordgo.MessageSend{
	Content: "We just have some basic questions for you.\nHow did you find us?",
	Components: []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "A member recruited me",
					CustomID: "onboarding:choice:recruited",
				},
				discordgo.Button{
					Label:    "Found Sol Armada on RSI",
					CustomID: "onboarding:choice:rsi",
				},
				discordgo.Button{
					Label:    "Some other way",
					CustomID: "onboarding:choice:other",
				},
				discordgo.Button{
					Label:    "Just visiting",
					CustomID: "onboarding:choice:visiting",
				},
			},
		},
	},
}

func OnboardingCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logging := log.WithField("command", "Onboarding")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		logging.WithError(err).Error("getting command user for permissions")
		handlers.ErrorResponse(s, i.Interaction, "internal server error")
		return
	}

	storage := stores.Storage
	u := &user.User{}
	if err := storage.GetUser(i.Member.User.ID).Decode(u); err != nil {
		logging.WithError(err).Error("getting command user for permissions")
		handlers.ErrorResponse(s, i.Interaction, "internal server error")
		return
	}

	if u.Rank > ranks.Lieutenant {
		handlers.ErrorResponse(s, i.Interaction, "You don't have permission for this command.")
		return
	}

	m, err := s.GuildMember(i.GuildID, i.ApplicationCommandData().Options[0].Value.(string))
	if err != nil {
		logging.WithError(err).Error("getting guild member")
		return
	}

	if _, ok := choiceMade[m.User.ID]; ok {
		choiceMade[m.User.ID] = ""
	}

	if _, err := s.GuildMemberEditComplex(i.GuildID, m.User.ID, &discordgo.GuildMemberParams{
		Roles: &[]string{},
		Nick:  m.User.Username,
	}); err != nil {
		logging.WithError(err).Error("reverting user")
		return
	}

	// Update the notification thread
	onboardingChannelId := config.GetString("DISCORD.CHANNELS.ONBOARDING")
	messages, err := s.ChannelMessages(onboardingChannelId, 100, "", "", "")
	if err != nil {
		logging.WithError(err).Error("getting all onboarding notification messages")
		return
	}

	for _, message := range messages {
		if strings.Contains(message.Content, m.User.Username) {
			if message.Thread != nil {
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s is re-onboarding %s", u.Name, m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread for re-onboarding")
					return
				}
				break
			}
			if err := s.ChannelMessageDelete(message.ChannelID, message.ID); err != nil {
				logging.WithError(err).Error("deleting onboarding notification message")
				return
			}
			break
		}
	}

	// remove all currently onboarding chanels for this user
	channels, err := s.GuildChannels(i.GuildID)
	if err != nil {
		logging.WithError(err).Error("getting all channels")
		return
	}

	for _, c := range channels {
		if c.Name == fmt.Sprintf("onboarding-%s", strings.ToLower(strings.ReplaceAll(m.User.Username, " ", "-"))) {
			if _, err := s.ChannelDelete(c.ID); err != nil {
				logging.WithError(err).Error("deleting old onboarding channel")
				return
			}
		}
	}

	onboarding(s, m)
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: fmt.Sprintf("Started onboarding process for %s", m.User.Username),
	}); err != nil {
		logging.WithError(err).Error("interaction response")
	}
}

func ChoiceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "ChoiceButtonHandler")
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		logger.WithError(err).Error("getting channel")
		return
	}

	onboardingUser := strings.Replace(channel.Name, "onboarding-", "", 1)
	if onboardingUser != strings.ToLower(strings.ReplaceAll(i.Member.User.Username, " ", "-")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "This onboarding process does not belong to you.",
			},
		}); err != nil {
			logger.WithError(err).Error("returning miss matched onboarding user")
			return
		}

		return
	}

	choiceMade[i.Member.User.ID] = strings.Split(i.MessageComponentData().CustomID, ":")[2]

	if choiceMade[i.Member.User.ID] == "visiting" {
		airlockChannelId := config.GetStringWithDefault("DISCORD.CHANNELS.AIRLOCK", "")
		airlockName := "#airlock"
		if airlockChannelId != "" {
			airlockChannel, err := s.Channel(airlockChannelId)
			if err != nil {
				logger.WithError(err).Error("getting airlock channel")
				return
			}

			airlockName = airlockChannel.Mention()
		}
		if _, err := s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("Okay! If you change your mind and would like to join Sol Armada, please contact an @Officer in the %s", airlockName)); err != nil {
			logger.WithError(err).Error("returning miss matched onboarding user")
			handlers.ErrorResponse(s, i.Interaction, "Something happened in the backend. I am notifying the admins now. Please standby. Use this channel if you need any other assistance.")
			return
		}

		finish(s, i)
		return
	}

	questions := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "rsi_handle",
					Label:       "Your RSI handle",
					Style:       discordgo.TextInputShort,
					Placeholder: "Your handle can be found on your public RSI page",
					Required:    true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "play_time",
					Label:       "How long have you been playing SC?",
					Style:       discordgo.TextInputShort,
					Placeholder: "Example: 2 years or 1 month",
					Required:    true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "gameplay",
					Label:       "What gameplay are you most interested in?",
					Style:       discordgo.TextInputShort,
					Placeholder: "Combat, Rescue, Mining, etc",
					Required:    true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "age",
					Label:    "How old are you?",
					Style:    discordgo.TextInputShort,
					Required: true,
				},
			},
		},
	}

	if choiceMade[i.Member.User.ID] == "recruited" {
		questions = append(questions,
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						Label:       "Who recruited you?",
						CustomID:    "who_recruited",
						Style:       discordgo.TextInputShort,
						Placeholder: "The recruiter's handle",
						Required:    true,
					},
				},
			})
	}

	if choiceMade[i.Member.User.ID] == "other" {
		questions = append(questions,
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "other",
						Label:    "How did you find us?",
						Style:    discordgo.TextInputParagraph,
						Required: true,
					},
				},
			})
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "onboarding:rsi_handle:" + i.Interaction.Member.User.ID,
			Title:      "What is your RSI handle?",
			Components: questions,
		},
	}); err != nil {
		logger.WithError(err).Error("responding to choice")
	}
}

func RSIModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	if !rsi.ValidHandle(value) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I couldn't find that RSI handle!\n\nPlease make sure it is correct and try again.\nYour RSI handle can be found on your public RSI profile page or in your settings here: https://robertsspaceindustries.com/account/settings",
			},
		}); err != nil {
			log.WithError(err).Error("RSI modal handler")
		}

		startOver(s, i.Interaction)
		return
	}

	finish(s, i)
}

func StartOverHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "StartOver")
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
		Data: &discordgo.InteractionResponseData{
			Content: "",
		},
	}); err != nil {
		logger.WithError(err).Error("interaction response")
	}

	messages, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting channel messages")
	}

	for _, message := range messages {
		if !strings.Contains(message.Content, "Welcome") && !strings.Contains(message.Content, "How did you find us?") {
			if err := s.ChannelMessageDelete(i.ChannelID, message.ID); err != nil {
				logger.WithError(err).Error("deleting channel messages")
			}
		}
	}
}

func notifyOfOnboarding(s *discordgo.Session, m *discordgo.Member) {
	logger := log.WithField("func", "notifyOfOnboarding")

	channelID := config.GetString("DISCORD.CHANNELS.ONBOARDING")

	messages, err := s.ChannelMessages(channelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting all onboarding notification messages")
		return
	}

	for _, message := range messages {
		if strings.Contains(message.Content, m.User.Username) {
			return
		}
	}
	if _, err := s.ChannelMessageSend(channelID, fmt.Sprintf("Onboarding %s", m.User.Username)); err != nil {
		log.WithError(err).Error("sending onboarding message")
		return
	}
}

func onboarding(s *discordgo.Session, m *discordgo.Member) {
	logger := log.WithField("func", "onboarding")
	notifyOfOnboarding(s, m)
	newChannel, err := s.GuildChannelCreateComplex("997836773927428156", discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("onboarding-%s", strings.ToLower(strings.ReplaceAll(m.User.Username, " ", "-"))),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.GetString("DISCORD.CATEGORIES.ONBOARDING"),
		Topic:    "Onboarding and Help",
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:    m.User.ID,
				Type:  discordgo.PermissionOverwriteTypeMember,
				Allow: 68672,
				Deny:  0,
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("creating a channel")
	}

	if _, err := s.ChannelMessageSend(newChannel.ID, fmt.Sprintf("Welcome, %s!", m.User.Mention())); err != nil {
		logger.WithError(err).Error("sending message")
	}

	time.Sleep(3 * time.Second)

	if _, err := s.ChannelMessageSendComplex(newChannel.ID, basicQuestions); err != nil {
		logger.WithError(err).Error("sending messsage with buttons")
	}
}

func startOver(s *discordgo.Session, i *discordgo.Interaction) {
	logger := log.WithField("func", "startOver")
	logger.Debug("starting over")
	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Would you like to try again?\nYou can come back here at any time and start over. You can also message in this channel if you need help!",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: fmt.Sprintf("onboarding:start_over:%s", i.Member.User.ID),
						Label:    "Yes! Let's start over",
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("sending message with buttons")
	}
}

func disableQuestionButtons(s *discordgo.Session, i *discordgo.Interaction) {
	logger := log.WithField("func", "disableQuestionButtons")
	messages, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting channel messages")
	}

	var message *discordgo.Message
	for _, m := range messages {
		if strings.Contains(m.Content, "How did you find us?") {
			message = m
			break
		}
	}

	for index, bq := range message.Components[0].(*discordgo.ActionsRow).Components {
		modified := bq.(*discordgo.Button)
		modified.Disabled = true
		message.Components[0].(*discordgo.ActionsRow).Components[index] = modified
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         message.ID,
		Channel:    i.Message.ChannelID,
		Components: message.Components,
	}); err != nil {
		logger.WithError(err).Error("editing message")
	}
}

func finish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "finish")

	disableQuestionButtons(s, i.Interaction)

	choice := choiceMade[i.Member.User.ID]

	airlockChannelId := config.GetStringWithDefault("DISCORD.CHANNELS.AIRLOCK", "")
	airlockName := "#airlock"
	if airlockChannelId != "" {
		airlockChannel, err := s.Channel(airlockChannelId)
		if err != nil {
			logger.WithError(err).Error("getting airlock channel")
			return
		}

		airlockName = airlockChannel.Mention()
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("If you have any other questions, please ask an @Officer for help in the %s!", airlockName),
		},
	}); err != nil {
		logger.WithError(err).Error("sending first finish message")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}

	// notify of completed onboarding
	onboardingChannelID := config.GetString("DISCORD.CHANNELS.ONBOARDING")
	messages, err := s.ChannelMessages(onboardingChannelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting onboarding messages")
		return
	}

	// get the original message for notification thread
	originalThreadMessage := &discordgo.Message{}
	for _, m := range messages {
		if strings.Contains(m.Content, i.Member.User.Username) {
			originalThreadMessage = m
			break
		}
	}

	// if they are just a visitor, move on
	if choice == "visiting" {
		thread, err := s.MessageThreadStartComplex(onboardingChannelID, originalThreadMessage.ID, &discordgo.ThreadStart{
			Name:                "Onboarding",
			AutoArchiveDuration: 60,
			Invitable:           false,
			RateLimitPerUser:    10,
		})
		if err != nil {
			logger.WithError(err).Error("creating onboarding thread")
			return
		}
		if _, err := s.ChannelMessageSend(thread.ID, fmt.Sprintf("%s is a visitor", i.Member.User.Username)); err != nil {
			logger.WithError(err).Error("creating onboarding thread")
			handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
			return
		}
		go removeChannelMessage(s, i, logger)
		return
	}

	// not a visitor
	data := i.ModalSubmitData()
	RSIHandle := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	playTime := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	gameplay := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	age := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	recruiter := ""
	found := "Via RSI"

	if len(data.Components) >= 5 {
		if data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).CustomID == "other" {
			found = data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		} else {
			recruiter = data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		}
	}

	po, affiliatedOrgs, _, err := rsi.GetOrgInfo(RSIHandle)
	if err != nil {
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue when getting your RSI information. Please try again in a little bit.")
		return
	}

	thread := originalThreadMessage.Thread
	if thread == nil {
		thread, err = s.MessageThreadStartComplex(onboardingChannelID, originalThreadMessage.ID, &discordgo.ThreadStart{
			Name:                fmt.Sprintf("%s finished onboarding", i.Member.User.Username),
			AutoArchiveDuration: 60,
			Invitable:           false,
			RateLimitPerUser:    10,
		})
		if err != nil {
			logger.WithError(err).Error("creating onboarding thread")
			return
		}
	}

	em := &discordgo.MessageEmbed{
		Type:  discordgo.EmbedTypeArticle,
		Title: "Information",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "RSI Handle",
				Value: RSIHandle,
			},
			{
				Name:  "RSI Profile",
				Value: fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", RSIHandle),
			},
			{
				Name:  "Primary Org",
				Value: po,
			},
			{
				Name:  "Affiliate Orgs",
				Value: strings.Join(affiliatedOrgs, ", "),
			},
			{
				Name:  "Playtime",
				Value: playTime,
			},
			{
				Name:  "Gameplay",
				Value: gameplay,
			},
			{
				Name:  "Age",
				Value: age,
			},
		},
	}

	if recruiter != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Recruiter",
			Value: recruiter,
		})
	}

	if found != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "How they found us",
			Value: found,
		})
	}

	if _, err := s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			em,
		},
	}); err != nil {
		logger.WithError(err).Error("sending onboarding thread message")
		return
	}

	// finish up
	if _, err := s.ChannelMessageSend(
		i.ChannelID,
		"Here is the link to our handbook: https://handbook.solarmada.space/ (redirects to a Google Doc)",
	); err != nil {
		logger.WithError(err).Error("sending message")
	}

	if _, err := s.ChannelMessageSend(
		i.ChannelID,
		"You can submit an application, if you have not already, to Sol Armada at: https://join.solarmada.space/ (redirects to an RSI page)",
	); err != nil {
		logger.WithError(err).Error("sending message")
	}

	// setup to update the member
	params := &discordgo.GuildMemberParams{
		Nick: RSIHandle,
	}

	// attach a role if we have one
	roleId := config.GetStringWithDefault("DISCORD.ROLE_IDS.GUEST", "")
	if roleId != "" {
		params.Roles = &[]string{roleId}
	}

	// update their nick
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, params); err != nil {
		logger.WithError(err).Error("editing the member")
		handlers.ErrorResponse(s, i.Interaction, "Something happened in the backend. I am notifying the admins now. Please standby. Use this channel if you need any other assistance.")
		return
	}

	go removeChannelMessage(s, i, logger)
}

func removeChannelMessage(s *discordgo.Session, i *discordgo.InteractionCreate, logger *log.Entry) {
	message, err := s.ChannelMessageSend(
		i.ChannelID,
		"This channel will be removed in about 30 minutes.",
	)
	if err != nil {
		logger.WithError(err).Error("sending removed channel message")
	}

	for i := 29; i > 0; i-- {
		time.Sleep(1 * time.Minute)
		if _, err := s.ChannelMessageEdit(
			message.ChannelID,
			message.ID,
			fmt.Sprintf("This channel will be removed in about %d minutes.", i),
		); err != nil {
			if strings.Contains(err.Error(), strconv.Itoa(discordgo.ErrCodeUnknownChannel)) {
				return
			}

			logger.WithError(err).Error("editing message")
		}
	}

	if _, err := s.ChannelDelete(i.ChannelID); err != nil {
		logger.WithError(err).Error("deleting channel")
	}
}
