package event

import (
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/kyokomi/emoji/v2"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

func EventReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	logger := log.WithField("handler", "EventReactionAdd")
	// ignore all reactions created by bots
	if r.UserID == s.State.User.ID {
		return
	}

	if r.ChannelID != config.GetString("DISCORD.CHANNELS.EVENTS") {
		return
	}

	event, err := events.GetByMessageId(r.MessageID)
	if err != nil {
		logger.WithError(err).Error("event reaction add")
		return
	}
	if event == nil {
		return
	}

	for _, position := range event.Positions {
		positionEmoji := emoji.CodeMap()[":"+strings.ToLower(position.Emoji)+":"]

		// the intended position reaction
		if r.Emoji.Name == positionEmoji {
			// see if the position is full
			if len(position.Members) == int(position.Max) {
				if err := s.MessageReactionRemove(r.ChannelID, r.MessageID, positionEmoji, r.UserID); err != nil {
					logger.WithError(err).Error("removing reaction from event message")
				}

				return
			}

			// add them to the position
			position.Members = append(position.Members, r.UserID)

			continue
		}

		// if not the inteded position reaction

		// update the members to exclude this member from
		// the other positions
		members := []string{}
		for _, memberInPos := range position.Members {
			if memberInPos != r.UserID {
				members = append(members, memberInPos)
			}
		}
		position.Members = members

		// remove their reaction
		if err := s.MessageReactionRemove(r.ChannelID, r.MessageID, positionEmoji, r.UserID); err != nil {
			logger.WithError(err).Error("removing reaction from event message")
			return
		}
	}

	if err := event.Save(); err != nil {
		logger.WithError(err).Error("saving event")
		return
	}

	if err := updateEventMessage(s, event); err != nil {
		logger.WithError(err).Error("updating event message")
		return
	}
}
