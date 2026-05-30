package bot

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
)

func onLeaveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	slog.Debug("member left")

	if m.User.Bot {
		return
	}

	member, err := members.Get(m.User.ID)
	if err != nil && !errors.Is(err, members.MemberNotFound) {
		slog.Error("getting member", "error", err)
		return
	}

	if err := member.Delete("Left the server"); err != nil {
		slog.Error("deleting member", "error", err)
		return
	}

	memberMessage := member.GetOnboardingMessage()
	if _, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: member.ChannelId,
		ID:      member.MessageId,
		Content: &memberMessage.Content,
		Embeds:  &memberMessage.Embeds,
	}); err != nil {
		slog.Error("editing member message on leave", "error", err)
	}
}
