package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func endEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

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
		Embeds:     &attendanceMessage.Embeds,
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

	if len(fromStartMembers) >= 0 && attendance.Tokenable {
		msg, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
		if err != nil {
			return err
		}

		ch := msg.Thread
		if msg.Thread == nil {
			ch, err = s.MessageThreadStartComplex(attendance.ChannelId, attendance.MessageId, &discordgo.ThreadStart{
				Name:                "Attendance for " + attendance.Name,
				Type:                discordgo.ChannelTypeGuildPublicThread,
				Invitable:           true,
				AutoArchiveDuration: 1440,
			})
			if err != nil {
				return err
			}
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

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.ToPointer("Event ended!"),
	})
	return err
}
