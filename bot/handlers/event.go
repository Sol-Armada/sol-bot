package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

var eventSubCommands = map[string]func(*discordgo.Session, *discordgo.Interaction){
	"attendance": takeAttendance,
}

var activeEvent *discordgo.Message

func EventCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// get the user
	storage := stores.Storage
	userResault := storage.GetUser(i.User.ID)
	user := &users.User{}
	if err := userResault.Decode(user); err != nil {
		errorResponse(s, i.Interaction, "Internal server error... >_<; Try again later")
		return
	}

	// check for permission
	if user.Rank > ranks.Lieutenant {
		errorResponse(s, i.Interaction, "You do not have permission to use this command")
		return
	}

	// send to the sub command
	if handler, ok := eventSubCommands[i.ApplicationCommandData().Options[0].Name]; ok {
		handler(s, i.Interaction)
		return
	}

	// somehow they used a sub command that doesn't exist
	errorResponse(s, i.Interaction, "That sub command doesn't exist. Not sure how you even got here. Good job.")
}

func takeAttendance(s *discordgo.Session, i *discordgo.Interaction) {
	g, err := s.State.Guild(i.GuildID)
	if err != nil {
		log.WithError(err).Error("getting guild state")
		return
	}

	if len(g.VoiceStates) == 0 {
		if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "No one is in voice chats!",
			},
		}); err != nil {
			log.WithError(err).Error("responding to take attendance command interaction")
		}
		return
	}

	// get the configured attendance channel id, otherwise use the channel we are in
	attendanceChannel, err := s.Channel(config.GetStringWithDefault("discord.channels.attendance", i.ChannelID))
	if err != nil {
		log.WithError(err).Error("getting attendance channel")
		return
	}

	if activeEvent != nil {
		content := "There is already an event being tracked."
		if i.ChannelID != attendanceChannel.ID {
			content += fmt.Sprintf(" Check %s", attendanceChannel.Mention())
		}
		if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: content,
			},
		}); err != nil {
			log.WithError(err).Error("responding to take attendance command interaction")
		}
		return
	}

	// this is for testing purposes only
	// g.VoiceStates = []*discordgo.VoiceState{}
	// for i := 0; i < 20; i++ {
	// 	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321")
	// 	randomString := make([]rune, 20)
	// 	for i := range randomString {
	// 		randomString[i] = chars[rand.Intn(len(chars))]
	// 	}

	// 	g.VoiceStates = append(g.VoiceStates, &discordgo.VoiceState{
	// 		Member: &discordgo.Member{
	// 			User: &discordgo.User{
	// 				ID:       fmt.Sprint(rand.Intn(999999999)),
	// 				Username: string(randomString),
	// 			},
	// 			Nick: "",
	// 		},
	// 	})
	// }

	rows := []discordgo.ActionsRow{}
	buttons := []discordgo.MessageComponent{}
	rowIndex := 0
	for _, vs := range g.VoiceStates {
		label := vs.Member.User.Username
		if vs.Member.Nick != "" {
			label = vs.Member.Nick
		}
		if len(label) >= 20 {
			label = label[:17] + "..."
		}
		buttons = append(buttons, discordgo.Button{
			Label:    label,
			CustomID: "event:attendance:toggle:" + vs.Member.User.ID,
			Style:    discordgo.PrimaryButton,
		})
		if len(buttons) < 5 {
			continue
		}

		rows = append(rows, discordgo.ActionsRow{
			Components: buttons,
		})
		buttons = []discordgo.MessageComponent{}
		rowIndex++
	}

	content := "Taking attendance..."
	if i.ChannelID != attendanceChannel.ID {
		content += fmt.Sprintf(" check out %s", attendanceChannel.Mention())
	}
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: content,
		},
	}); err != nil {
		log.WithError(err).Error("responding to take attendance command interaction")
	}
	rows = append(rows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Submit",
				Style:    discordgo.SuccessButton,
				CustomID: "event:attendance:submit",
			},
		},
	})
	components := []discordgo.MessageComponent{}
	for _, r := range rows {
		components = append(components, r)
	}
	m, err := s.ChannelMessageSendComplex(attendanceChannel.ID, &discordgo.MessageSend{
		Content:    "Click any member to toggle their attendance.\n:blue_square: attended    :red_square: not attended",
		Components: components,
	})
	if err != nil {
		log.WithError(err).Error("sending message to channel for attendance command")
	}

	activeEvent = m
}

func EventInteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if strings.HasPrefix(i.MessageComponentData().CustomID, "event:attendance:toggle:") {
		toggleAttendance(s, i.Interaction)
		return
	}
	if i.MessageComponentData().CustomID == "event:attendance:submit" {
		submitAttendance(s, i.Interaction)
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "That sub command doesn't exist. Not sure how you even got here. Good job.",
		},
	}); err != nil {
		log.WithError(err).Error("responding to event command interaction")
	}
}

func toggleAttendance(s *discordgo.Session, i *discordgo.Interaction) {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.WithError(err).Error("responding to toggle attendance interaction")
		return
	}

	memberButtonToToggle := i.MessageComponentData().CustomID

	for _, row := range i.Message.Components {
		r := row.(*discordgo.ActionsRow)

		for index, component := range r.Components {
			if component.Type() == discordgo.ButtonComponent {
				c := component.(*discordgo.Button)
				if c.CustomID == memberButtonToToggle {
					if c.Style == discordgo.PrimaryButton {
						c.Style = discordgo.DangerButton
					} else {
						c.Style = discordgo.PrimaryButton
					}

					r.Components[index] = c
					break
				}
			}
		}
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         i.Message.ID,
		Channel:    i.Message.ChannelID,
		Components: i.Message.Components,
	}); err != nil {
		log.WithError(err).Error("editing original attendance message")
	}
}

func submitAttendance(s *discordgo.Session, i *discordgo.Interaction) {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.WithError(err).Error("responding to submit attendance interaction")
	}

	// get the configured attendance channel id, otherwise use the channel we are in
	attendanceChannel, err := s.Channel(config.GetStringWithDefault("discord.channels.attendance", i.ChannelID))
	if err != nil {
		log.WithError(err).Error("getting attendance channel")
		return
	}

	attendies := ""
	for _, row := range i.Message.Components {
		r := row.(*discordgo.ActionsRow)
		for _, button := range r.Components {
			b := button.(*discordgo.Button)
			if b.Style == discordgo.PrimaryButton {
				attendies += b.Label + "\n"
			}
		}
	}
	if _, err := s.ChannelMessageSendComplex(attendanceChannel.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("%s\n%s", time.Now().Format("**Jan 02**"), strings.TrimRight(attendies, "\n")),
	}); err != nil {
		log.WithError(err).Error("sending attendance sumbittion message")
		return
	}

	if err := s.ChannelMessageDelete(activeEvent.ChannelID, activeEvent.ID); err != nil {
		log.WithError(err).Error("deleting original attendance message")
	}

	activeEvent = nil
}

func errorResponse(s *discordgo.Session, i *discordgo.Interaction, message string) {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: message,
		},
	}); err != nil {
		log.WithError(err).Error("responding to event command interaction")
	}
}
