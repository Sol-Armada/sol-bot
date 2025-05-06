package giveaway

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/utils"
)

func (g *Giveaway) UpdateInputs() error {
	_, err := g.sess.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelId,
		ID:         g.InputMessageId,
		Components: utils.ToPointer(g.GetComponents()),
	})
	return err
}
