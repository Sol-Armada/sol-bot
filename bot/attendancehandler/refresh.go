package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func refreshButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("refresh button handler")

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	if err := a.RecheckIssues(); err != nil {
		return errors.Wrap(err, "rechecking issues for attendance record")
	}

	message, err := a.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: a.ChannelId,
		ID:      a.MessageId,
		Content: &message.Content,
		Embeds:  &message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "editing attendance message for rechecking issues")
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
