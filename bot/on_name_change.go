package bot

import (
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/activity"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/utils"
)

func OnNameChangeHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	slog.Debug("member name changed")

	if m.User.Bot {
		return
	}

	member, err := members.Get(m.User.ID)
	if err != nil {
		slog.Error("getting member", "error", err)
		return
	}

	member.Name = m.User.Username
	if err := member.Save(); err != nil {
		slog.Error("saving member", "error", err)
		return
	}

	member.UpdateRoles(m.Member.Roles)

	if err := rsi.UpdateMemberRSIInfo(member, &utils.ExponentialBackoff{
		MaxRetries: 3,
		Multiplier: 1.1,
		MaxDelay:   time.Second,
	}, slog.Default()); err != nil {
		slog.Error("updating RSI info", "error", err)
		return
	}

	memberMessage := member.GetOnboardingMessage()
	if _, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: member.ChannelId,
		ID:      member.MessageId,
		Content: &memberMessage.Content,
		Embeds:  &memberMessage.Embeds,
	}); err != nil {
		slog.Error("editing member message on name change", "error", err)
	}

	nameUpdateActivity := &activity.Activity{
		Who:  member,
		When: time.Now().UTC(),
		Meta: activity.Meta{
			What: activity.VoiceJoin,
			Where: map[string]string{
				"old": m.Member.Nick,
				"new": m.User.Username,
			},
		},
	}
	if err := nameUpdateActivity.Save(); err != nil {
		slog.Error("saving name change activity", "error", err)
	}
}
