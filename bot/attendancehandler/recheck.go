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

func recheckIssuesButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("rechecking issues button handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	if err := attendance.RecheckIssues(); err != nil {
		return errors.Wrap(err, "rechecking issues for attendance record")
	}

	message := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: attendance.ChannelId,
		ID:      attendance.MessageId,
		Content: &message.Content,
		Embeds:  &message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "editing attendance message for rechecking issues")
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Attendance record <#%s> rechecked", attendance.MessageId),
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}
