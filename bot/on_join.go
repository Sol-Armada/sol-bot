package bot

import (
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
)

func onJoinHandler(s *discordgo.Session, i *discordgo.GuildMemberAdd) {
	logger := log.WithFields(log.Fields{
		"guild":   i.GuildID,
		"user":    i.User.ID,
		"handler": "OnJoinHandler",
	})

	logger.Info("member joined")

	if i.Member.User.Bot {
		return
	}

	member := members.New(i.Member)

	var err error
	member, err = rsi.UpdateRsiInfo(member)
	if err != nil && !errors.Is(err, rsi.RsiUserNotFound) {
		logger.WithError(err).Error("updating rsi info")
	}

	if err := member.Save(); err != nil {
		logger.WithError(err).Error("saving member")
		return
	}

	if settings.GetString("FEATURES.ONBOARDING.CHANNEL_ID") != "" {
		onBoardingMessage := member.GetOnboardingMessage()

		message, err := s.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.CHANNEL_ID"), &discordgo.MessageSend{
			Content: onBoardingMessage.Content,
			Embeds:  onBoardingMessage.Embeds,
		})
		if err != nil {
			logger.WithError(err).Error("sending onboarding message")
			return
		}

		member.ChannelId = message.ChannelID
		member.MessageId = message.ID

		if err := member.Save(); err != nil {
			logger.WithError(err).Error("saving member after onboarding message")
		}
	}
}
