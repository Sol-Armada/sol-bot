package giveaway

import (
	"github.com/bwmarrin/discordgo"
)

func (g *Giveaway) UpdateMessage(s *discordgo.Session) error {
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: g.ChannelId,
		ID:      g.EmbedMessageId,
		Embeds:  &[]*discordgo.MessageEmbed{g.GetEmbed()},
	})
	return err
}
