package onboarding

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
)

type processing struct {
	Handle    string
	Choice    string
	Found     string
	PlayTime  string
	GamePlay  string
	Age       string
	Recruiter string
	MessageId string
}

var processingUsers map[string]*processing = map[string]*processing{}

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

	delete(processingUsers, m.User.ID)

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

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: fmt.Sprintf("Started onboarding process for %s", m.User.Username),
	}); err != nil {
		logging.WithError(err).Error("interaction response")
	}
}

func ChoiceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "ChoiceButtonHandler")

	airlockChannelId := config.GetStringWithDefault("DISCORD.CHANNELS.AIRLOCK", "")
	airlockName := "#airlock"
	if airlockChannelId != "" {
		airlockName = fmt.Sprintf("<#%s>", airlockChannelId)
	}

	if len(i.Member.Roles) > 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You have already been onboarded",
			},
		}); err != nil {
			logger.WithError(err).Error("messaging user that they are already onboarded")
			return
		}

		return
	}

	processingUsers[i.Member.User.ID] = &processing{
		Choice: strings.Split(i.MessageComponentData().CustomID, ":")[2],
	}

	if processingUsers[i.Member.User.ID].Choice == "visiting" {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Okay! If you change your mind and would like to join Sol Armada, please contact an @Officer in the %s", airlockName),
			},
		}); err != nil {
			logger.WithError(err).Error("interaction response - visiting user")
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

	if processingUsers[i.Member.User.ID].Choice == "recruited" {
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

	if processingUsers[i.Member.User.ID].Choice == "other" {
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
			Title:      "Some questions about you",
			Components: questions,
		},
	}); err != nil {
		logger.WithError(err).Error("responding to choice")
	}
}

func RSIModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	processingUsers[i.Member.User.ID].Handle = data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// if we are comming from he original modal
	if len(data.Components) > 1 {
		processingUsers[i.Member.User.ID].PlayTime = data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		processingUsers[i.Member.User.ID].GamePlay = data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		processingUsers[i.Member.User.ID].Age = data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

		processingUsers[i.Member.User.ID].Found = "Via RSI"

		if len(data.Components) >= 5 {
			if data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).CustomID == "other" {
				processingUsers[i.Member.User.ID].Found = data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			} else {
				processingUsers[i.Member.User.ID].Recruiter = data.Components[4].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
			}
		}
	}

	if !rsi.ValidHandle(processingUsers[i.Member.User.ID].Handle) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "I couldn't find that RSI handle!\n\nPlease make sure it is correct and try again.\nYour RSI handle can be found on your public RSI profile page or in your settings here: https://robertsspaceindustries.com/account/settings",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Try Again",
								CustomID: "onboarding:try_rsi_handle_again:" + i.Interaction.Member.User.ID,
							},
						},
					},
				},
			},
		}); err != nil {
			log.WithError(err).Error("RSI modal handler")
		}

		return
	}

	finish(s, i)
}

func TryAgainHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "TryAgainHandler")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "onboarding:rsi_handle:" + i.Interaction.Member.User.ID,
			Title:    "Some questions about you",
			Components: []discordgo.MessageComponent{
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
			},
		},
	}); err != nil {
		logger.WithError(err).Error("responding to choice")
	}
}

func finish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "finish")

	airlockChannelId := config.GetStringWithDefault("DISCORD.CHANNELS.AIRLOCK", "")
	airlockName := "#airlock"
	if airlockChannelId != "" {
		airlockName = fmt.Sprintf("<#%s>", airlockChannelId)
	}

	rulesChannelId := config.GetStringWithDefault("DISCORD.CHANNELS.RULES", "")
	rulesName := "#rules"
	if rulesChannelId != "" {
		rulesName = fmt.Sprintf("<#%s>", rulesChannelId)
	}

	// notify of completed onboarding
	onboardingChannelID := config.GetString("DISCORD.CHANNELS.ONBOARDING")
	var originalThreadMessage *discordgo.Message
	messages, err := s.ChannelMessages(onboardingChannelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting all onboarding notification messages")
		return
	}

	for _, message := range messages {
		if strings.Contains(message.Content, i.Member.User.Username) {
			originalThreadMessage = message
			break
		}
	}

	// if they are just a visitor, move on
	if processingUsers[i.Member.User.ID].Choice == "visiting" {
		if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Flags:    discordgo.MessageFlagsEphemeral,
			Content:  fmt.Sprintf("If you have any other questions, please ask an @Officer for help in the %s!\n\nReturn here if you change your mind about joining Sol Armada!", airlockName),
			Username: i.Member.User.Username,
		}); err != nil {
			logger.WithError(err).Error("sending followup message")
			handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
			return
		}

		if _, err := s.ChannelMessageSend(originalThreadMessage.Thread.ID, fmt.Sprintf("%s is a visitor", i.Member.User.Username)); err != nil {
			logger.WithError(err).Error("creating onboarding thread")
			handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend.")
			return
		}
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		logger.WithError(err).Error("sending first finish message")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}

	if originalThreadMessage == nil {
		originalThreadMessage, err = s.ChannelMessageSend(onboardingChannelID, fmt.Sprintf("Onboarding %s", i.Member.User.Username))
		if err != nil {
			log.WithError(err).Error("sending onboarding message")
			return
		}
		if _, err := s.MessageThreadStartComplex(onboardingChannelID, originalThreadMessage.ID, &discordgo.ThreadStart{
			Name:                "Onboarding",
			AutoArchiveDuration: 60,
			Invitable:           false,
			RateLimitPerUser:    10,
		}); err != nil {
			log.WithError(err).Error("starting thread on onboarding message")
			return
		}
	}

	// not a visitor
	po, affiliatedOrgs, _, err := rsi.GetOrgInfo(processingUsers[i.Member.User.ID].Handle)
	if err != nil {
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue when getting your RSI information. Please try again in a little bit.")
		return
	}

	em := &discordgo.MessageEmbed{
		Type:  discordgo.EmbedTypeArticle,
		Title: "Information",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "RSI Handle",
				Value: processingUsers[i.Member.User.ID].Handle,
			},
			{
				Name:  "RSI Profile",
				Value: fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", processingUsers[i.Member.User.ID].Handle),
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
				Value: processingUsers[i.Member.User.ID].PlayTime,
			},
			{
				Name:  "Gameplay",
				Value: processingUsers[i.Member.User.ID].GamePlay,
			},
			{
				Name:  "Age",
				Value: processingUsers[i.Member.User.ID].Age,
			},
		},
	}

	if processingUsers[i.Member.User.ID].Recruiter != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Recruiter",
			Value: processingUsers[i.Member.User.ID].Recruiter,
		})
	} else {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "How they found us",
			Value: processingUsers[i.Member.User.ID].Found,
		})
	}

	id := originalThreadMessage.ID
	if originalThreadMessage.Thread != nil {
		id = originalThreadMessage.Thread.ID
	}
	if _, err := s.ChannelMessageSendComplex(id, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			em,
		},
	}); err != nil {
		logger.WithError(err).Error("sending onboarding thread message")
		return
	}

	// finish up
	// setup to update the member
	params := &discordgo.GuildMemberParams{
		Nick: processingUsers[i.Member.User.ID].Handle,
	}

	// attach a role if we have one
	roleId := config.GetStringWithDefault("DISCORD.ROLE_IDS.ONBOARDED", "")
	if roleId != "" {
		params.Roles = &[]string{roleId}
	}

	// update their nick
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, params); err != nil {
		logger.WithError(err).Error("editing the member")
		handlers.ErrorResponse(s, i.Interaction, "Something happened in the backend. I am notifying the admins now. Please standby. Use this channel if you need any other assistance.")
		return
	}

	delete(processingUsers, i.Member.User.ID)

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Flags:    discordgo.MessageFlagsEphemeral,
		Content:  fmt.Sprintf("If you have any other questions, please ask an @Officer for help in the %s!\n\nRead our %s and enjoy the Sol Armada community!", airlockName, rulesName),
		Username: i.Member.User.Username,
	}); err != nil {
		logger.WithError(err).Error("sending followup message")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}
}
