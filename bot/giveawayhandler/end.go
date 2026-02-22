package giveawayhandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/utils"
)

func end(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if !utils.Allowed(i.Member, "GIVEAWAYS") {
		return customerrors.InvalidPermissions
	}

	giveawayId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	g := giveaway.GetGiveaway(giveawayId)
	if err := g.End(); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Content: "Giveaway ended",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
