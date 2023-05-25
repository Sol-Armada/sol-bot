package event

import (
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/kyokomi/emoji/v2"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

func EventReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	logger := log.WithField("handler", "EventReactionRemove")
	// ignore all reactions created by bots
	if r.UserID == s.State.User.ID {
		return
	}

	if r.ChannelID != config.GetString("DISCORD.CHANNELS.EVENTS") {
		return
	}

	event, err := events.GetByMessageId(r.MessageID)
	if err != nil {
		logger.WithError(err).Error("getting event")
		return
	}
	if event == nil {
		return
	}

	event.Lock()
	for _, position := range event.Positions {
		positionEmoji := emoji.CodeMap()[strings.ToLower(position.Emoji)]

		if r.Emoji.Name == positionEmoji {
			membersInPos := []string{}
			for _, memberId := range position.Members {
				if memberId != r.UserID {
					membersInPos = append(membersInPos, memberId)
				}
			}
			position.Members = membersInPos
		}
	}
	event.Unlock()

	if err := event.Save(); err != nil {
		logger.WithError(err).Error("saving event")
		return
	}

	if err := updateEventMessage(s, event); err != nil {
		logger.WithError(err).Error("updating event")
		return
	}
}
