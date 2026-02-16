package rafflehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func cancel(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle cancel button")

	if !utils.Allowed(i.Member, "RAFFLES") {
		_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "You do not have the permissions to do that.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	data := i.MessageComponentData()

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return err
	}

	if err := raffle.Delete(); err != nil {
		return err
	}

	if err := s.ChannelMessageDelete(i.Interaction.ChannelID, i.Message.ID); err != nil {
		return err
	}

	return nil
}
