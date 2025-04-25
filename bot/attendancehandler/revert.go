package attendancehandler

import (
	"context"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func revertAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	if data.Options[0].Options[0].Focused {
		attendanceRecords, err := attdnc.List(bson.D{}, 10, 1)
		if err != nil {
			return errors.Wrap(err, "getting recorded attendance records")
		}

		for _, record := range attendanceRecords {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  record.Name,
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

func revertCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("reverting attendance command handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData()
	id := data.Options[0].Options[0].StringValue()

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	msg, _ := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Reverting attendance...",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	if err := attendance.Revert(); err != nil {
		return errors.Wrap(err, "reverting attendance")
	}

	attendanceMessage, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
	if err != nil {
		if derr, ok := err.(*discordgo.RESTError); ok {
			if derr.Response.StatusCode == 404 {
				_, _ = s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
					Content: utils.ToPointer("It looks like that attendance record message is missing! Creating it again..."),
				})

				if _, err := s.ChannelMessageSendComplex(attendance.ChannelId, &discordgo.MessageSend{
					Content:    "Recreated because message was missing!",
					Embeds:     attendance.ToDiscordMessage().Embeds,
					Components: attendance.ToDiscordMessage().Components,
				}); err != nil {
					return err
				}

				return nil
			}
		}

		return errors.Wrap(err, "getting attendance message")
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendanceMessage.ChannelID,
		ID:         attendanceMessage.ID,
		Embeds:     &attendance.ToDiscordMessage().Embeds,
		Components: &attendance.ToDiscordMessage().Components,
	}); err != nil {
		return err
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
		Content: utils.ToPointer("Attendance reverted!"),
	})

	return nil
}
