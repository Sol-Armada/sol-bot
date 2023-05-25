package event

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

func updateEventMessage(s *discordgo.Session, event *events.Event) error {
	// get the event message
	message, err := s.ChannelMessage(config.GetString("DISCORD.CHANNELS.EVENTS"), event.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting event message")
	}

	embeds, err := event.GetEmbeds()
	if err != nil {
		return errors.Wrap(err, "getting embeds")
	}

	if _, err := s.ChannelMessageEditEmbeds(message.ChannelID, message.ID, []*discordgo.MessageEmbed{
		embeds,
	}); err != nil {
		return errors.Wrap(err, "updating embeds")
	}

	return nil
}
