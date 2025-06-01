package attendancehandler

import (
	"context"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

func exportButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("add member attendance command")

	data := i.Interaction.MessageComponentData()

	eventId := strings.Split(data.CustomID, ":")[2]
	attendance, err := attdnc.Get(eventId)
	if err != nil {
		if errors.Is(err, attdnc.ErrAttendanceNotFound) {
			return customerrors.InvalidAttendanceRecord
		}

		return errors.Wrap(err, "getting attendance record")
	}

	members := attendance.GetMembers(true)
	if len(members) == 0 {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No members to export.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return errors.Wrap(err, "responding to interaction")
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Exporting attendance data...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	thread := i.Message.Thread
	if thread == nil {
		thread, _ = s.MessageThreadStartComplex(i.Message.ChannelID, i.Message.ID, &discordgo.ThreadStart{
			Name:                "Attendance Thread - " + attendance.Name + " (" + attendance.DateCreated.Format("2006-01-02 15:04:05") + ")",
			Type:                discordgo.ChannelTypeGuildPublicThread,
			AutoArchiveDuration: 60,
		})
		_ = s.ThreadMemberAdd(thread.ID, i.Member.User.ID)
	}

	sb := strings.Builder{}
	sb.WriteString("Here is the list of members who attended the event: " + attendance.Name + "\n")
	sb.WriteString("```\n")
	for i, member := range members {
		n := member.Name
		if len(members)-1 != i {
			n += ","
		}
		sb.WriteString(n)
	}
	sb.WriteString("\n```")

	_, err = s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Content: sb.String(),
	})

	return err
}
