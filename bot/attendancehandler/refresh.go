package attendancehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func refreshCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("refreshing attendance command handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if lastRefreshTime.After(time.Now().Add(-1 * time.Hour)) {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Already refreshed in the last hour!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return nil
	}

	attendance, err := attdnc.List(bson.D{}, 10, 1)
	if err != nil {
		return errors.Wrap(err, "getting attendance records")
	}

	m, _ := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Attendance refreshing...",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	for idx, a := range attendance {
		msg := a.ToDiscordMessage()
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    a.ChannelId,
			ID:         a.MessageId,
			Embeds:     &msg.Embeds,
			Components: &msg.Components,
		}); err != nil {
			return err
		}

		_, _ = s.FollowupMessageEdit(i.Interaction, m.ID, &discordgo.WebhookEdit{
			Content: utils.ToPointer(fmt.Sprintf("Attendance refreshing... (%d/%d)", idx+1, len(attendance))),
		})

		time.Sleep(250 * time.Millisecond)
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, m.ID, &discordgo.WebhookEdit{
		Content: utils.ToPointer("Attendance refreshed!"),
	})

	lastRefreshTime = time.Now()

	return nil
}
