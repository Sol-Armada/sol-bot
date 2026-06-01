package jobs

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func promotionsReport(ctx context.Context, s *discordgo.Session, monitor JobMonitor) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("promotions report job")

	monitor.Update("Preparing promotions report")

	channelId := settings.GetString("PROMOTIONS_CHANNEL_ID")
	if channelId == "" {
		return fmt.Errorf("promotions channel id not set")
	}

	monitor.Update("Building promotions report")
	embed, err := members.GetPromotionsEmbed()
	if err != nil {
		if customerrors.Is(err, customerrors.NoPromotions) {
			monitor.Update("No promotions to report")
			return nil
		}
		return err
	}

	monitor.Update("Sending promotions report")
	_, err = s.ChannelMessageSendEmbed(channelId, embed)
	return err
}
