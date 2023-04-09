package onboarding

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/config"
)

func JoinServerHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	logging := log.WithField("handler", "OnJoin:Onboarding")

	// Update the notification thread
	onboardingChannelId := config.GetString("DISCORD.CHANNELS.ONBOARDING")
	messages, err := s.ChannelMessages(onboardingChannelId, 100, "", "", "")
	if err != nil {
		logging.WithError(err).Error("getting all onboarding notification messages")
		return
	}

	for _, message := range messages {
		if strings.Contains(message.Content, m.User.Username) {
			if message.Thread != nil {
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s has re-joined the server", m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread for re-onboarding")
				}
			}
			return
		}
	}

	message, err := s.ChannelMessageSend(config.GetString("DISCORD.CHANNELS.ONBOARDING"), m.User.Username+" joined")
	if err != nil {
		log.WithError(err).Error("on join onboarding")
		return
	}

	if _, err := s.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
		Name:                "Onboarding",
		AutoArchiveDuration: 60,
		Invitable:           false,
		RateLimitPerUser:    10,
	}); err != nil {
		log.WithError(err).Error("starting thread on onboarding message")
	}
}
