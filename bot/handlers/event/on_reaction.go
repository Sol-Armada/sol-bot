package event

import (
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

func EventReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.ChannelID != config.GetString("DISCORD.CHANNELS.EVENTS") {
		return
	}

	e, err := events.GetByMessageId(r.MessageID)
	if err != nil {
		log.WithError(err).Error("event reaction add")
		return
	}

	_ = e
}
