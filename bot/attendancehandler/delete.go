package attendancehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func deleteButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("deleting attendance button handler")

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance == nil {
		_ = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Looks like that attendance doesn't exist in the database anyway, removed the message.",
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
}

func verifyDeleteButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("deleting verify handler")

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance != nil {
		if err := attendance.Delete(); err != nil {
			return errors.Wrap(err, "deleting attendance record")
		}

		msg, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
		if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
			return errors.Wrap(err, "getting attendance message")
		}

		if msg.Thread != nil {
			if err := s.ChannelMessageDelete(msg.Thread.ID, msg.Thread.ID); err != nil {
				derr := err.(*discordgo.RESTError)
				if derr.Response.StatusCode != 404 {
					return errors.Wrap(err, "deleting attendance thread")
				}
			}
		}

		if err := s.ChannelMessageDelete(attendance.ChannelId, attendance.MessageId); err != nil {
			derr := err.(*discordgo.RESTError)
			if derr.Response.StatusCode != 404 {
				return errors.Wrap(err, "deleting attendance message")
			}
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Content:    "Attendance record deleted.",
			Components: []discordgo.MessageComponent{},
		},
	})
}

func cancelDeleteButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("deleting cancel handler")

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Content:    "Whew, that was close. Attendance record not deleted.",
			Components: []discordgo.MessageComponent{},
		},
	})
}
