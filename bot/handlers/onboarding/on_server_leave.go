package onboarding

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
)

func LeaveServerHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	logging := log.WithField("handler", "OnLeave")
	channels, err := s.GuildChannels(m.GuildID)
	if err != nil {
		logging.WithError(err).Error("getting all channels")
		return
	}

	for _, c := range channels {
		if c.Name == fmt.Sprintf("onboarding-%s", strings.ToLower(strings.ReplaceAll(m.User.Username, " ", "-"))) {
			if _, err := s.ChannelDelete(c.ID); err != nil {
				logging.WithError(err).Error("deleting old onboarding channel")
				return
			}
		}

		if c.ID == config.GetString("DISCORDGO.CHANNELS.ONBOARDING") {
			messages, err := s.ChannelMessages(c.ID, 100, "", "", "")
			if err != nil {
				logging.WithError(err).Error("getting messages in onboarding channel")
				return
			}

			for _, message := range messages {
				if strings.Contains(message.Content, m.User.Username) {
					if _, err := s.ChannelMessageEdit(message.ChannelID, message.ID, fmt.Sprintf("Onboarding %s (left the server)", m.User.Username)); err != nil {
						logging.WithError(err).Error("updating onboarding thread message")
						return
					}
				}
			}
		}
	}
}
