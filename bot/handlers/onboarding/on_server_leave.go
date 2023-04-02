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
				if _, err := s.ChannelMessageSend(message.Thread.ID, fmt.Sprintf("%s has left the server", m.User.Username)); err != nil {
					logging.WithError(err).Error("replying to thread on leave")
					return
				}
				break
			}
			if err := s.ChannelMessageDelete(message.ChannelID, message.ID); err != nil {
				logging.WithError(err).Error("deleting onboarding notification message")
				return
			}
			break
		}
	}
}
