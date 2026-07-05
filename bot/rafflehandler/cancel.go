package rafflehandler

import (
	"context"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func cancel(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle cancel button")

	data := i.MessageComponentData()

	raffleIdStr := strings.Split(data.CustomID, ":")[2]

	raffleId, err := strconv.Atoi(raffleIdStr)
	if err != nil {
		return err
	}

	raffle, err := raffles.Get(int32(raffleId))
	if err != nil {
		return err
	}

	if err := raffle.Delete(); err != nil {
		return err
	}

	if err := s.ChannelMessageDelete(i.Interaction.ChannelID, i.Message.ID); err != nil {
		return err
	}

	return s.InteractionResponseDelete(i.Interaction)
}
