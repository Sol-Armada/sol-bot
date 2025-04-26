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

	g := giveaway.GetGiveaway(giveawayId).End()

	_ = s.ChannelMessageUnpin(g.ChannelId, g.EmbedMessageId)

	if _, err := s.ChannelMessageSendComplex(g.ChannelId, &discordgo.MessageSend{
		Components: g.GetComponents(),
		Embeds:     []*discordgo.MessageEmbed{g.GetEmbed()},
	}); err != nil {
		return err
	}

	_ = s.ChannelMessageDelete(g.ChannelId, g.EmbedMessageId)
	_ = s.ChannelMessageDelete(g.ChannelId, g.InputMessageId)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Content: "Giveaway ended",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
