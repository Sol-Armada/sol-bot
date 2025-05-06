package giveawayhandler

import (
	"context"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/utils"
)

func exit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("giveaway exit button")

	giveawayId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	g := giveaway.GetGiveaway(giveawayId)
	g.RemoveMemberFromItems(i.Member.User.ID)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Removed you from the giveaway",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
