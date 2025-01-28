package attendancehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

func deleteButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting attendance button handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance == nil {
		_ = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Looks like that attendance doesn't exist in the database anyway, removed the message.",
			},
		})
		return nil
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Are you sure you want to delete this attendance record?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Yes",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("attendance:verifydelete:%s", id),
						},
						discordgo.Button{
							Label:    "No",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("attendance:canceldelete:%s", id),
						},
					},
				},
			},
		},
	})

	return nil
}

func verifyDeleteButtonModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting verify modal handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return customerrors.InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance != nil {
		if err := attendance.Delete(); err != nil {
			return errors.Wrap(err, "deleting attendance record")
		}

		if err := s.ChannelMessageDelete(attendance.ChannelId, attendance.MessageId); err != nil {
			derr := err.(*discordgo.RESTError)
			if derr.Response.StatusCode != 404 {
				return errors.Wrap(err, "deleting attendance message")
			}
		}
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content:    utils.StringPointer("Attendance record deleted!"),
		Components: &[]discordgo.MessageComponent{},
	})

	return nil
}

func cancelDeleteButtonModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting cancel modal handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Whew, that was close. Attendance record not deleted.",
		},
	})

	return nil
}
