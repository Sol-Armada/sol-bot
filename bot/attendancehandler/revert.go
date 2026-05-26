package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func revertAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	if data.Options[0].Options[0].Focused {
		attendanceRecords, err := attdnc.List(nil, 10, 1)
		if err != nil {
			return errors.Wrap(err, "getting recorded attendance records")
		}

		for _, record := range attendanceRecords {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  record.Name + " (" + record.Id + ")",
				Value: record.Id,
			})
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to revert attendance auto complete")
	}

	return nil
}

func revertButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("reverting attendance command handler")

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	if err := attendance.Revert(); err != nil {
		return errors.Wrap(err, "reverting attendance")
	}

	attendanceMessage, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
	if err != nil {
		if derr, ok := err.(*discordgo.RESTError); ok {
			if derr.Response.StatusCode == 404 {
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Somehow you tried to revert an attendance record that doesn't exist. Impressive.",
				})
				return err
			}
		}
		return errors.Wrap(err, "getting attendance message")
	}

	m, err := attendance.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendanceMessage.ChannelID,
		ID:         attendanceMessage.ID,
		Embeds:     &m.Embeds,
		Components: &m.Components,
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
	})
}
