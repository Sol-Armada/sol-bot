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
	if i.Message.Thread == nil {
		thread, _ = s.ThreadStartComplex(i.ChannelID, &discordgo.ThreadStart{
			Name: "Attendance Thread - " + attendance.Name + " (" + attendance.DateCreated.Format("2006-01-02 15:04:05") + ")",
			Type: discordgo.ChannelTypeGuildPublicThread,
		})
		_ = s.ThreadMemberAdd(thread.ID, i.Member.User.ID)
	}

	csvData := ""
	for _, member := range members {
		csvData += member.Name + "\n"
	}
	csvReader := strings.NewReader(csvData)

	_, err = s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Content: "Here is the list of members who attended the event: " + attendance.Name,
		Flags:   discordgo.MessageFlagsEphemeral,
		Files: []*discordgo.File{
			{
				Name:   "attendance-" + attendance.Id + ".csv",
				Reader: csvReader,
			},
		},
	})

	return err
}
