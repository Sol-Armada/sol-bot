package onboarding

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	customerrors "github.com/sol-armada/admin/errors"
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
	User      *user.User
}

var processingUsers map[string]*processing = map[string]*processing{}
var validations map[string]string = map[string]string{}

// onboarding handlers
var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice":               choiceButtonHandler,
	"try_rsi_handle_again": tryAgainHandler,
	"validate":             validateButtonHandler,
}
var modalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"rsi_handle": rsiModalHandler,
}
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"onboarding": onboardingCommandHandler,
	"validate":   validateCommandHandler,
}

func Setup(b *bot.Bot) error {
	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			id := strings.Split(i.MessageComponentData().CustomID, ":")
			if id[0] == "onboarding" {
				interactionHandlers[id[1]](s, i)
			}
		case discordgo.InteractionModalSubmit:
			id := strings.Split(i.ModalSubmitData().CustomID, ":")
			if id[0] == "onboarding" {
				modalHandlers[id[1]](s, i)
			}
		}
	})
	// watch for server join
	b.AddHandler(joinServerHandler)
	// watch for server leave
	b.AddHandler(leaveServerHandler)

	if err := setupChannel(b); err != nil {
		return errors.Wrap(err, "failed to setup onboarding channel")
	}

	// commands
	if err := b.DeleteCommand("join"); err != nil {
		log.WithError(err).Error("unable to delete join command")
	}
	if err := b.DeleteCommand("onboarding"); err != nil {
		log.WithError(err).Error("unable to delete onboarding command")
	}

	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "onboarding",
		Description: "Start onboarding process for someone",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "member",
				Description:  "the member to onboard",
				Type:         discordgo.ApplicationCommandOptionMentionable,
				Required:     true,
				Autocomplete: true,
			},
		},
	}); err != nil {
		return errors.Wrap(err, "creating oboarding command")
	}

	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "validate",
		Description: "Validate your RSI profile",
		Type:        discordgo.ChatApplicationCommand,
	}); err != nil {
		return errors.Wrap(err, "creating validate command")
	}

	return nil
}

func setupChannel(b *bot.Bot) error {
	var oc *discordgo.Channel
	channels, err := b.GuildChannels(b.GuildId)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.ParentID == config.GetString("DISCORD.CATEGORIES.AIRLOCK") && channel.Name == "onboarding" {
			oc = channel
			break
		}
	}

	m := `Welcome to Sol Armada!
	
Select a reason you joined below. We will ask a few questions then assign you a role.`
	components := []discordgo.MessageComponent{
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
	}

	if oc == nil {
		oc, err = b.GuildChannelCreateComplex(b.GuildId, discordgo.GuildChannelCreateData{
			Name:     "onboarding",
			Type:     discordgo.ChannelTypeGuildText,
			ParentID: config.GetString("DISCORD.CATEGORIES.AIRLOCK"),
		})
		if err != nil {
			return err
		}

		if _, err := b.ChannelMessageSendComplex(oc.ID, &discordgo.MessageSend{
			Content:    m,
			Components: components,
		}); err != nil {
			return err
		}

		return nil
	}

	messages, err := b.ChannelMessages(oc.ID, 100, "", "", "")
	if err != nil {
		return err
	}

	if len(messages) != 0 {
		if _, err := b.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content:    &m,
			Components: components,

			ID:      messages[0].ID,
			Channel: oc.ID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func onboardingCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logging := log.WithField("command", "Onboarding")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		logging.WithError(err).Error("getting command user for permissions")
		return
	}

	storage := stores.Storage
	u := &user.User{}
	if err := storage.GetUser(i.Member.User.ID).Decode(u); err != nil {
		logging.WithError(err).Error("getting command user for permissions")
		return
	}

	if u.Rank > ranks.Lieutenant {
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

	found := false
	for _, message := range messages {
		if strings.Contains(message.Content, m.User.Username) {
			if message.Thread != nil {
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s is re-onboarding %s", u.Name, m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread for re-onboarding")
					return
				}

				break
			}

			found = true
			break
		}
	}

	if !found {
		onboardingMessage, err := s.ChannelMessageSend(config.GetString("DISCORD.CHANNELS.ONBOARDING"), m.User.Username+" joined")
		if err != nil {
			log.WithError(err).Error("on join onboarding")
			return
		}

		if _, err := s.MessageThreadStartComplex(onboardingMessage.ChannelID, onboardingMessage.ID, &discordgo.ThreadStart{
			Name:                "Re-onboarding",
			AutoArchiveDuration: 60,
			Invitable:           false,
			RateLimitPerUser:    10,
		}); err != nil {
			log.WithError(err).Error("starting thread on onboarding message")
			return
		}
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: fmt.Sprintf("Started onboarding process for %s", m.User.Username),
	}); err != nil {
		logging.WithError(err).Error("interaction response")
	}
}

func choiceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	u := user.New(i.Member)
	if err := u.Save(); err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "There was an error! Try again in a little few minutes or let the @Officers know")
		return
	}
	processingUsers[i.Member.User.ID].User = u

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
			customerrors.ErrorResponse(s, i.Interaction, "Something happened in the backend. I am notifying the admins now. Please standby. Use this channel if you need any other assistance.")
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

func rsiModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	processingUsers[i.Member.User.ID].Handle = data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	// if we are comming from he original modal
	if len(data.Components) > 1 {
		processingUsers[i.Member.User.ID].PlayTime = data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
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

func tryAgainHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			return
		}

		if _, err := s.ChannelMessageSend(originalThreadMessage.Thread.ID, fmt.Sprintf("%s is a visitor", i.Member.User.Username)); err != nil {
			logger.WithError(err).Error("creating onboarding thread")
			customerrors.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend.")
			return
		}
		return
	}

	// send a wait while we process everything else
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		logger.WithError(err).Error("sending first finish message")
		customerrors.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}

	// create the notification thread if we don't have one
	if originalThreadMessage == nil {
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
	u := processingUsers[i.Member.User.ID].User
	u, err = rsi.GetOrgInfo(u)
	if err != nil {
		customerrors.ErrorResponse(s, i.Interaction, "Ran into an issue when getting your RSI information. Please try again in a little bit.")
		return
	}

	age, err := strconv.ParseInt(processingUsers[i.Member.User.ID].Age, 10, 32)
	if err != nil {
		log.WithError(err).Error("could not parse age")
	}
	u.Age = int(age)
	u.Gameplay = processingUsers[i.Member.User.ID].GamePlay
	u.Playtime = processingUsers[i.Member.User.ID].PlayTime
	if err := u.Save(); err != nil {
		log.WithError(err).Error("could not save user finishing onboarding")
	}

	// build the notification embed
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
				Value: u.PrimaryOrg,
			},
			{
				Name:  "Affiliate Orgs",
				Value: strings.Join(u.Affilations, ", "),
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
		customerrors.ErrorResponse(s, i.Interaction, "Something happened in the backend. I am notifying the Officers now. Please standby. Use the airlock if you need any other assistance.")
		return
	}

	delete(processingUsers, i.Member.User.ID)

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Flags:    discordgo.MessageFlagsEphemeral,
		Content:  fmt.Sprintf("Your name has been updated to your RSI handle.\n\nIf you have any other questions, please ask an @Officer for help in the %s!\n\nIf you have not already, please check out our handbook and see if you want to join Sol Aramada.\n\nHandbook: https://handbook.solarmada.space/\nJoin us: https://join.solarmada.space/\n\nRead our %s and enjoy the Sol Armada community!", airlockName, rulesName),
		Username: i.Member.User.Username,
	}); err != nil {
		logger.WithError(err).Error("sending followup message")
		customerrors.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}
}

func joinServerHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	logging := log.WithFields(log.Fields{
		"handler": "OnJoin:Onboarding",
		"member":  m,
	})
	logging.Info("someone joined")

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
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s has re-joined the server", m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread for re-onboarding")
					return
				}
			}
			return
		}
	}

	message, err := s.ChannelMessageSend(config.GetString("DISCORD.CHANNELS.ONBOARDING"), m.User.Username+" joined")
	if err != nil {
		log.WithError(err).Error("on join onboarding")
		return
	}

	if _, err := s.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
		Name:                "Onboarding",
		AutoArchiveDuration: 60,
		Invitable:           false,
	}); err != nil {
		log.WithError(err).Error("starting thread on onboarding message")
	}
}

func leaveServerHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	logging := log.WithField("handler", "OnLeave")

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
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s has left the server", m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread on leave")
					return
				}

				break
			}

			break
		}
	}
}

func generateValidationCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	src := rand.NewSource(time.Now().UnixNano())

	r := rand.New(src)

	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func validateCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	code := generateValidationCode()
	validations[i.Member.User.ID] = code

	u, err := user.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user for validation")
		customerrors.ErrorResponse(s, i.Interaction, "")
		return
	}

	if u.Validated {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are already validated! Congrats!",
			},
		}); err != nil {
			log.WithError(err).Error("validated response")
			customerrors.ErrorResponse(s, i.Interaction, "")
			return
		}
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Please temporarily place the below generated code anywhere in your RSI profile bio. Then click the \"Validate\" button.\n\n%s", code),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Validate",
							CustomID: fmt.Sprintf("onboarding:validate:%s", i.Member.User.ID),
						},
					},
				},
			},
		},
	}); err != nil {
		log.WithError(err).Error("responding to validate command")
		customerrors.ErrorResponse(s, i.Interaction, "Ran into an error! Try again in a few minutes or let the @Officers know")
		return
	}
}

func validateButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]
	code, ok := validations[id]
	if !ok {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "It looks like I am not prepared to validate your profile. Please the command again or let the @Officers know if you keep running into this message.",
			},
		}); err != nil {
			log.WithError(err).Error("responding to validate button")
			customerrors.ErrorResponse(s, i.Interaction, "")
		}
		return
	}

	u, err := user.Get(id)
	if err != nil {
		log.WithError(err).Error("getting user for validation")
		customerrors.ErrorResponse(s, i.Interaction, "")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.WithError(err).Error("deffered message response for validation")
		customerrors.ErrorResponse(s, i.Interaction, "")
		return
	}

	go func() {
		waitTime := time.Duration(2 * time.Second)
		ticker := time.NewTicker(waitTime)

		attempt := 1
		for {
			bio, err := rsi.GetBio(u.GetTrueNick())
			if err != nil {
				log.WithError(err).Error("getting rsi profile for validation")
				customerrors.ErrorResponse(s, i.Interaction, "")
				return
			}

			if !strings.Contains(bio, code) {
				if attempt == 3 {
					if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "I could not find the code on your profile. Please try the command again and give a minute after adding the code to your bio before clicking \"Validate\"",
					}); err != nil {
						log.WithError(err).Error("responding to failed validation")
						customerrors.ErrorResponse(s, i.Interaction, "")
					}
					ticker.Stop()
					return
				}

				goto CONTINUE
			}

			ticker.Stop()
			delete(validations, id)
			break

		CONTINUE:
			attempt++
			ticker.Reset(waitTime)
			<-ticker.C
		}

		u.Validated = true
		if err := u.Save(); err != nil {
			log.WithError(err).Error("saving user for validation")
			customerrors.ErrorResponse(s, i.Interaction, "")
		}

		if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Your account has been validated! You can remove the code from you bio.",
		}); err != nil {
			log.WithError(err).Error("creating follow up message")
			customerrors.ErrorResponse(s, i.Interaction, "")
			return
		}
	}()
}
