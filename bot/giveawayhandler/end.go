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

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelId,
		ID:         g.MessageId,
		Components: utils.ToPointer(g.GetComponents()),
		Embeds:     &[]*discordgo.MessageEmbed{g.GetEmbed()},
	}); err != nil {
		return err
	}

	return nil
}
