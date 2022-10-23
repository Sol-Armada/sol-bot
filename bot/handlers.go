package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/rsi"
)

var choiceMade map[string]string = map[string]string{}

func JoinServerHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	newChannel, err := s.GuildChannelCreateComplex("997836773927428156", discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("%s-onboarding", m.User.Username),
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
		panic(err)
	}

	if _, err := s.ChannelMessageSend(newChannel.ID, fmt.Sprintf("Welcome, %s!", m.User.Mention())); err != nil {
		panic(err)
	}

	time.Sleep(3 * time.Second)

	if _, err := s.ChannelMessageSendComplex(newChannel.ID, &discordgo.MessageSend{
		Content: "We just have some basic questions for you.\nWhy did you join our server?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Guest of an Org Member",
						CustomID: "choice:guest",
					},
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
		panic(err)
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
		panic(err)
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
			panic(err)
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
		panic(err)
	}
}

func StartOverHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
		Data: &discordgo.InteractionResponseData{
			Content: "",
		},
	}); err != nil {
		panic(err)
	}

	messages, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		panic(err)
	}

	for _, message := range messages {
		if !strings.Contains(message.Content, "Welcome") {
			if err := s.ChannelMessageDelete(i.ChannelID, message.ID); err != nil {
				panic(err)
			}
		}
	}

	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Why did you join our server?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Guest of an Org Member",
						CustomID: "choice:guest",
					},
					discordgo.Button{
						Label:    "Ally of the Org",
						CustomID: "choice:ally",
					},
					discordgo.Button{
						Label:    "Joining the Org",
						CustomID: "choice:join",
					},
				},
			},
		},
	}); err != nil {
		panic(err)
	}
}

func GuestButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	roleId := config.GetString("DISCORD.ROLE_IDS.GUEST")
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, &discordgo.GuildMemberParams{
		Nick: value,
		Roles: &[]string{
			roleId,
		},
	}); err != nil {
		panic(err)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Great! Your server nickname has been updated to match your RSI handle and you have been given Guest access!",
		},
	}); err != nil {
		panic(err)
	}

	disableQuestionButtons(s, i.Interaction)
	finish(s, i)
}

func GuestFriendOfModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		panic(err)
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
			panic(err)
		}
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponsePong,
	}); err != nil {
		panic(err)
	}

	finish(s, i)
}

func GuestFriendHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID); err != nil {
		panic(err)
	}
	finish(s, i)
}

func AllyButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fmt.Println("ally")

	if _, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: "Great!",
	}); err != nil {
		panic(err)
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
		panic(err)
	}
}

func AllyOrgModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			panic(err)
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
		panic(err)
	}

	finish(s, i)
}

func JoinButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	value := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	roleId := config.GetString("DISCORD.ROLE_IDS.RECRUIT")
	if _, err := s.GuildMemberEditComplex(i.GuildID, i.Member.User.ID, &discordgo.GuildMemberParams{
		Nick: value,
		Roles: &[]string{
			roleId,
		},
	}); err != nil {
		panic(err)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Great! Your server nickname has been updated to match your RSI handle and you have been given Recruit access!",
		},
	}); err != nil {
		panic(err)
	}

	disableQuestionButtons(s, i.Interaction)
	finish(s, i)
}

func disableQuestionButtons(s *discordgo.Session, i *discordgo.Interaction) {
	messages, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		panic(err)
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
					discordgo.Button{
						Label:    "Guest of an Org Member",
						CustomID: "guest",
						Disabled: true,
					},
					discordgo.Button{
						Label:    "Ally of the Org",
						CustomID: "ally",
						Disabled: true,
					},
					discordgo.Button{
						Label:    "Joining the Org",
						CustomID: "join",
						Disabled: true,
					},
				},
			},
		},
	}); err != nil {
		panic(err)
	}
}

func finish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, err := s.ChannelMessageSend(i.ChannelID, "Awesome!"); err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	message, err := s.ChannelMessageSend(
		i.ChannelID,
		"Thank you for answering our questions!\nThis channel will be removed in about 30 seconds",
	)
	if err != nil {
		panic(err)
	}

	for i := 29; i >= 0; i-- {
		time.Sleep(1 * time.Second)
		if _, err := s.ChannelMessageEdit(
			message.ChannelID,
			message.ID,
			fmt.Sprintf("Thank you for answering our questions!\nThis channel will be removed in about %d seconds", i),
		); err != nil {
			panic(err)
		}
	}

	if _, err := s.ChannelDelete(i.ChannelID); err != nil {
		panic(err)
	}
}
