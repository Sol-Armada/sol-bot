package raffles

import "github.com/bwmarrin/discordgo"

func (r *Raffle) UpdateMessage(s *discordgo.Session) error {
	embed, err := r.GetEmbed()
	if err != nil {
		return err
	}

	if r.Ended {
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    r.ChannelId,
			ID:         r.MessageId,
			Components: &[]discordgo.MessageComponent{},
		}); err != nil {
			return err
		}
	}

	_, err = s.ChannelMessageEditEmbed(r.ChannelId, r.MessageId, embed)
	return err
}
