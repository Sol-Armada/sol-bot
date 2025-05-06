package giveaway

import (
	"github.com/bwmarrin/discordgo"
)

func (g *Giveaway) UpdateMessage() error {
	_, err := g.sess.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: g.ChannelId,
		ID:      g.EmbedMessageId,
		Embeds:  &[]*discordgo.MessageEmbed{g.GetEmbed()},
	})
	return err
}
