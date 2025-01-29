package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func endEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	if err := attendance.Record(); err != nil {
		return err
	}

	attendanceMessage := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendance.ChannelId,
		ID:         attendance.MessageId,
		Components: &attendanceMessage.Components,
		Embed:      attendanceMessage.Embed,
	}); err != nil {
		return err
	}

	fromStartMembers := make([]discordgo.SelectMenuOption, 0, len(attendance.FromStart))

	for _, memberId := range attendance.FromStart {
		member, ok := attendance.GetMember(memberId)
		if !ok {
			continue
		}

		fromStartMembers = append(fromStartMembers, discordgo.SelectMenuOption{
			Label: member.Name,
			Value: member.Id,
		})
	}

	if len(fromStartMembers) >= 0 {
		ch, err := s.ThreadStartComplex(attendance.ChannelId, &discordgo.ThreadStart{
			Name:                "Attendance for " + attendance.Name,
			Type:                discordgo.ChannelTypeGuildPrivateThread,
			Invitable:           true,
			AutoArchiveDuration: 1440,
		})
		if err != nil {
			return err
		}

		if err := s.ThreadMemberAdd(ch.ID, i.Interaction.Member.User.ID); err != nil {
			return err
		}

		if _, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
			Content: "Please select the members that stayed for the event",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "attendance:stayed:" + attendance.Id,
							Options:     fromStartMembers,
							Placeholder: "",
							MaxValues:   len(fromStartMembers),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Submit",
							Style:    discordgo.SuccessButton,
							CustomID: "attendance:stayed_submit:" + attendance.Id,
						},
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{},
	})
}
