package bot

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
)

func onJoinHandler(s *discordgo.Session, i *discordgo.GuildMemberAdd) {
	logger = logger.With(
		slog.String("guild", i.GuildID),
		slog.String("user", i.User.ID),
		slog.String("handler", "OnJoinHandler"),
	)

	logger.Info("member joined")

	if i.Member.User.Bot {
		return
	}

	member := members.New(i.Member)

	if err := member.Save(); err != nil {
		logger.Error("saving member", "error", err)
		return
	}

	if settings.GetString("FEATURES.ONBOARDING.OUTPUT_CHANNEL_ID") != "" {
		onBoardingMessage := member.GetOnboardingMessage()

		message, err := s.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.OUTPUT_CHANNEL_ID"), &discordgo.MessageSend{
			Content: onBoardingMessage.Content,
			Embeds:  onBoardingMessage.Embeds,
		})
		if err != nil {
			logger.Error("sending onboarding message", "error", err)
			return
		}

		member.ChannelId = message.ChannelID
		member.MessageId = message.ID

		if err := member.Save(); err != nil {
			logger.Error("saving member after onboarding message", "error", err)
		}
	}
}
