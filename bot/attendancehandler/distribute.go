package attendancehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func distributeButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	fromStartMembers := make([]discordgo.SelectMenuOption, 0, len(a.FromStart))
	for _, id := range a.FromStart {
		member, ok := a.GetMember(id)
		if !ok {
			continue
		}

		fromStartMembers = append(fromStartMembers, discordgo.SelectMenuOption{
			Label: member.Name,
			Value: member.Id,
		})
	}

	if len(fromStartMembers) == 0 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "No members where at the start of the event, no tokens to distribute",
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("attendance:distribute:%s", attendanceId),
			Title:    "Distribute Tokens",
			Components: []discordgo.MessageComponent{
				discordgo.Label{
					Label: "Select members who where at the start",
					Component: discordgo.SelectMenu{
						CustomID:  fmt.Sprintf("attendance:distribute:%s", attendanceId),
						Options:   fromStartMembers,
						MaxValues: len(fromStartMembers),
					},
				},
			},
		},
	})
}

func distributeModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.ModalSubmitData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	selectComponent := i.ModalSubmitData().Components[0].(*discordgo.Label).Component.(*discordgo.SelectMenu)

	selectedMembers := make([]string, 0, len(selectComponent.Values))
	selectedMembers = append(selectedMembers, selectComponent.Values...)

	var content strings.Builder
	distributedTo, err := attendance.DistributeTokens(selectedMembers)
	if err != nil {
		return err
	}

	for _, msg := range distributedTo {
		fmt.Fprintf(&content, "\n%s", msg)
	}

	if content.Len() == 0 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
		})
	}

	msg, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
	if err != nil {
		return err
	}

	ch := msg.Thread
	if ch == nil {
		ch, err = s.MessageThreadStartComplex(attendance.ChannelId, attendance.MessageId, &discordgo.ThreadStart{
			Name:                "Thread for " + attendance.Name + " (" + attendance.Id + ")",
			Type:                discordgo.ChannelTypeGuildPublicThread,
			Invitable:           true,
			AutoArchiveDuration: 1440,
		})
		if err != nil {
			return err
		}
	}

	if _, err := s.ChannelMessageSend(ch.ID, content.String()); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
	})
}
