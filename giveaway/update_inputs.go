package giveaway

import (
	"github.com/bwmarrin/discordgo"
)

func (g *Giveaway) UpdateInputs() error {
	_, err := g.sess.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelId,
		ID:         g.InputMessageId,
		Components:new(g.GetComponents()),
	})
	return err
}
