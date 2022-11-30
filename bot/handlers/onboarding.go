package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

var choiceMade map[string]string = map[string]string{}

func JoinServerHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	onboarding(s, m.Member)
}

func onboarding(s *discordgo.Session, m *discordgo.Member) {
	logger := log.WithField("handler", "JoinServer")
	newChannel, err := s.GuildChannelCreateComplex("997836773927428156", discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("onboarding-%s", m.User.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: "997836773927428157",
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

	if _, err := s.ChannelMessageSendComplex(newChannel.ID, &discordgo.MessageSend{
		Content: "We just have some basic questions for you.\nWhy did you join our server?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					// discordgo.Button{
					// 	Label:    "Guest of an Org Member",
					// 	CustomID: "choice:guest",
					// },
					// discordgo.Button{
					// 	Label:    "Ally of the Org",
					// 	CustomID: "choice:ally",
					// },
					discordgo.Button{
						Label:    "Joining the Org",
						CustomID: "choice:join",
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("sending messsage with buttons")
	}
}

func OnboardingCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logging := log.WithField("command", "Onboarding")
	storage := stores.Storage
	u := &users.User{}
	if err := storage.GetUser(i.User.ID).Decode(u); err != nil {
		logging.WithError(err).Error("getting command user for permissions")
		errorResponse(s, i.Interaction, "internal server error")
		return
	}

	if u.Rank > ranks.Lieutenant {
		errorResponse(s, i.Interaction, "You don't have permission for this command.")
		return
	}

	m, err := s.GuildMember(i.GuildID, i.ApplicationCommandData().Options[0].Value.(string))
	if err != nil {
		logging.WithError(err).Error("getting guild member")
	}

	if _, ok := choiceMade[m.User.ID]; ok {
		choiceMade[m.User.ID] = ""
	}

	onboarding(s, m)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Started their onboarding process",
		},
	}); err != nil {
		logging.WithError(err).Error("interaction response")
	}
}

func ChoiceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "rsi_handle_" + i.Interaction.Member.User.ID,
			Title:    "What is your RSI handle?",
			Content:  "You can find your handle on your RSI profile page.",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "rsi_handle",
							Label:    "Your RSI handle",
							Style:    discordgo.TextInputShort,
							Required: true,
						},
					},
				},
			},
		},
	}); err != nil {
		log.WithError(err).Error("responding to choice")
	}

	choiceMade[i.Member.User.ID] = strings.Split(i.MessageComponentData().CustomID, ":")[1]
}

func RSIModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	if !rsi.ValidHandle(value) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I couldn't find that RSI handle!\nPlease make sure it is correct and try again.\nIt will be located on your RSI profile page",
				Embeds: []*discordgo.MessageEmbed{
					{
						Image: &discordgo.MessageEmbedImage{
							URL: "https://imgur.com/auqGvbc",
						},
					},
				},
			},
		}); err != nil {
			log.WithError(err).Error("RSI modal handler")
		}

		startOver(s, i.Interaction)
		return
	}

	switch choiceMade[i.Member.User.ID] {
	case "guest":
		GuestButtonHandler(s, i)
	case "join":
		JoinButtonHandler(s, i)
	}
}

func startOver(s *discordgo.Session, i *discordgo.Interaction) {
	logger := log.WithField("func", "startOver")
	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Looks like something went wrong! Would you like to try again?\nYou can come back here at any time and start over. You can also message in this channel if you need help",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: fmt.Sprintf("start_over:%s", i.Member.User.ID),
						Label:    "Yes! Let's start over",
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("sending message with buttons")
	}
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
		if !strings.Contains(message.Content, "Welcome") {
			if err := s.ChannelMessageDelete(i.ChannelID, message.ID); err != nil {
				logger.WithError(err).Error("deleting channel messages")
			}
		}
	}

	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Why did you join our server?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					// discordgo.Button{
					// 	Label:    "Guest of an Org Member",
					// 	CustomID: "choice:guest",
					// },
					// discordgo.Button{
					// 	Label:    "Ally of the Org",
					// 	CustomID: "choice:ally",
					// },
					discordgo.Button{
						Label:    "Joining the Org",
						CustomID: "choice:join",
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("sending the question")
	}
}

func GuestButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "GuestButton")
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	roleId := config.GetString("DISCORD.ROLE_IDS.GUEST")
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, &discordgo.GuildMemberParams{
		Nick: value,
		Roles: &[]string{
			roleId,
		},
	}); err != nil {
		logger.WithError(err).Error("editing the member")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Great! Your server nickname has been updated to match your RSI handle and you have been given Guest access!",
		},
	}); err != nil {
		logger.WithError(err).Error("interaction response")
	}

	disableQuestionButtons(s, i.Interaction)
	finish(s, i)
}

func GuestFriendOfModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "GuestFriendOfModal")
	data := i.ModalSubmitData()

	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		logger.WithError(err).Error("getting members")
	}
	memberNames := []string{}
	membersList := map[string]*discordgo.Member{}
	for _, member := range members {
		if member.User.ID != i.Member.User.ID {
			memberNames = append(memberNames, member.User.Username)
			membersList[member.User.Username] = member
		}
	}
	ranks := fuzzy.FindFold(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, memberNames)

	if len(ranks) > 1 {
		buttons := []discordgo.MessageComponent{}
		for _, r := range ranks {
			buttons = append(buttons, discordgo.Button{
				Label:    r,
				CustomID: fmt.Sprintf("guest_friend:%s", membersList[r].User.ID),
			})
		}
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: buttons,
					},
				},
			},
		}); err != nil {
			logger.WithError(err).Error("interaction response")
		}
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
	}); err != nil {
		logger.WithError(err).Error("interaction response")
	}

	finish(s, i)
}

func GuestFriendHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "GuestFriend")
	if err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID); err != nil {
		logger.WithError(err).Error("GuestFriend")
	}
	finish(s, i)
}

func AllyButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "AllyButton")

	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Great!",
	}); err != nil {
		logger.WithError(err).Error("getting channel messages")
	}

	time.Sleep(2 * time.Second)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "ally_org_" + i.Interaction.Member.User.ID,
			Title:    "What is your Org?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "org",
							Label:    "The org's handle: https://robertsspaceindustries.com/orgs/[orgshandle]",
							Style:    discordgo.TextInputShort,
							Required: true,
						},
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("interaction response")
	}
}

func AllyOrgModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "AllyOrgModal")
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	if !rsi.IsAllyOrg(value) {
		// let them know that is not an ally
		// make guest
		return
	}

	isMember, err := rsi.IsMemberOfOrg(i.Member.Nick, value)
	if err != nil {
		if !errors.Is(err, rsi.UserNotFound) {
			logger.WithError(err).Error("checking if is member of the given org")
		}

		// let them know that their profile was not found
		// restart from the beginning
		return
	}

	if !isMember {
		// let them know their profile doesn't show as part of that org
		// make guest
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
	}); err != nil {
		logger.WithError(err).Error("getting channel messages")
	}

	finish(s, i)
}

func JoinButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("handler", "JoinButton")
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	roleId := config.GetString("DISCORD.ROLE_IDS.RECRUIT")
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, &discordgo.GuildMemberParams{
		Nick: value,
		Roles: &[]string{
			roleId,
		},
	}); err != nil {
		logger.WithError(err).Error("editing member")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: `Great! Your server nickname has been updated to match your RSI handle and you have been given Recruit access!
			We encourage you to check out the #get-roles channel to add roles for professions that you are interested in when Star Citizen releases.
			These roles will notify you when those kinds of activities are happening!

			Our handbook
			https://handbook.solarmada.space/
			Make sure to join the org on RSI!
			https://join.solarmada.space/`,
		},
	}); err != nil {
		logger.WithError(err).Error("interaction response")
	}

	// professionsMessage, err := s.ChannelMessageSendEmbed(i.ChannelID, &discordgo.MessageEmbed{
	// 	Title:       "Professions",
	// 	Description: "What professions are you interested in when Star Citizen is released?",
	// 	Color:       15277667,
	// 	Fields: []*discordgo.MessageEmbedField{
	// 		{
	// 			Name:   "🚛 Haulers",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "♻️",
	// 			Value:  "Scavengers",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "⛏️",
	// 			Value:  "Miners",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🧭",
	// 			Value:  "Scouts (non-combat)",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🛟",
	// 			Value:  "SOS Emergency Responders",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🪖",
	// 			Value:  "FPS Marines",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🕵️",
	// 			Value:  "Bounty Hunters",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🚀",
	// 			Value:  "Light Fighters",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "👁️",
	// 			Value:  "Recon",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🏁",
	// 			Value:  "Racers",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "💪",
	// 			Value:  "Gunners",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "🧩",
	// 			Value:  "Adventurers",
	// 			Inline: true,
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	logger.WithError(err).Error("send embedded message")
	// }

	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🚛")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "♻️")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "⛏️")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🧭")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🛟")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🪖")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🕵️")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🚀")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "👁️")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🏁")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "💪")
	// s.MessageReactionAdd(i.ChannelID, professionsMessage.ID, "🧩")

	// if _, err := s.ChannelMessageSendEmbed(i.ChannelID, &discordgo.MessageEmbed{
	// 	Title:       "Professions",
	// 	Description: "What professions are you interested in when Star Citizen is released?",
	// 	Color:       15277667,
	// 	Fields: []*discordgo.MessageEmbedField{
	// 		{
	// 			Name:   "1️⃣",
	// 			Value:  "America/Pacific GMT -7",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "2️⃣",
	// 			Value:  "America/Mountain GMT -6",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "3️⃣",
	// 			Value:  "America/Central GMT -5",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "4️⃣",
	// 			Value:  "America/Eastern GMT -4",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "5️⃣",
	// 			Value:  "Europe/GMT +0",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "6️⃣",
	// 			Value:  "Europe/GMT +1",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "7️⃣",
	// 			Value:  "Europe/GMT +2",
	// 			Inline: true,
	// 		},
	// 		{
	// 			Name:   "8️⃣",
	// 			Value:  "Australia/GMT +10 or +11",
	// 			Inline: true,
	// 		},
	// 	},
	// }); err != nil {
	// 	logger.WithError(err).Error("send embedded message")
	// }

	disableQuestionButtons(s, i.Interaction)
	finish(s, i)
}

func disableQuestionButtons(s *discordgo.Session, i *discordgo.Interaction) {
	logger := log.WithField("func", "disableQuestionButtons")
	messages, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		logger.WithError(err).Error("getting channel messages")
	}

	messageID := ""
	for _, message := range messages {
		if strings.Contains(message.Content, "Why did you join our server?") {
			messageID = message.ID
			break
		}
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      messageID,
		Channel: i.Message.ChannelID,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					// discordgo.Button{
					// 	Label:    "Guest of an Org Member",
					// 	CustomID: "guest",
					// 	Disabled: true,
					// },
					// discordgo.Button{
					// 	Label:    "Ally of the Org",
					// 	CustomID: "ally",
					// 	Disabled: true,
					// },
					discordgo.Button{
						Label:    "Joining the Org",
						CustomID: "join",
						Disabled: true,
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("editing message")
	}
}

func finish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "finish")
	if _, err := s.ChannelMessageSend(i.ChannelID, "Great!"); err != nil {
		logger.WithError(err).Error("sending message")
	}

	time.Sleep(2 * time.Second)

	message, err := s.ChannelMessageSend(
		i.ChannelID,
		"This channel will be removed in about 30 minutes.\nIf you need to repeat this process, please ask for help in the #airlock",
	)
	if err != nil {
		logger.WithError(err).Error("sending message")
	}

	for i := 30; i > 0; i-- {
		time.Sleep(1 * time.Minute)
		if _, err := s.ChannelMessageEdit(
			message.ChannelID,
			message.ID,
			fmt.Sprintf("This channel will be removed in about %d minutes.\nIf you need to repeat this process, please ask for help in the #airlock", i),
		); err != nil {
			logger.WithError(err).Error("editing message")
		}
	}

	if _, err := s.ChannelDelete(i.ChannelID); err != nil {
		logger.WithError(err).Error("deleting channel")
	}
}
