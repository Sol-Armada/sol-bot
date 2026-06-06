package loothandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/rolls"
	"github.com/sol-armada/sol-bot/utils"
)

func end(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loot end handler")

	rollEventId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	rollEvent, err := rolls.GetEvent(rollEventId)
	if err != nil {
		return err
	}
	if rollEvent == nil {
		customerrors.ErrorResponse(s, i.Interaction, "Roll event not found", nil)
		return nil
	}

	if err := rollEvent.End(); err != nil {
		logger.Error("failed to end roll event", "error", err)
		return err
	}

	// TODO: Process winners and send final results

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Content: "Roll ended",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
