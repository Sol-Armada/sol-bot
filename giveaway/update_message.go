package giveaway

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/utils"
)

func (g *Giveaway) UpdateMessage(s *discordgo.Session) error {
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelId,
		ID:         g.MessageId,
		Embeds:     &[]*discordgo.MessageEmbed{g.GetEmbed()},
		Components: utils.ToPointer(g.GetComponents()),
	})
	return err
}
