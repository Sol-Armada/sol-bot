package bot

import (
	"errors"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/activity"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/utils"
)

func OnNameChangeHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	slog.Debug("member name changed", "user_id", m.User.ID, "old_name", m.Member.Nick, "new_name", m.BeforeUpdate.Nick)

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

	member.UpdateRank(m.Member.Roles)

	if err := (&utils.ExponentialBackoff{
		MaxRetries: 3,
		Multiplier: 1.1,
		MaxDelay:   time.Second,
	}).Execute(func() error {
		profile, err := rsi.GetRSIInfo(member.Name)
		if err != nil {
			return err
		}

		affiliations := make([]string, len(profile.Affiliation))
		for i, aff := range profile.Affiliation {
			affiliations[i] = aff.Name
		}

		return member.UpdateRsiInfo()
	}); err != nil && !errors.Is(err, rsi.ErrUserNotFound) {
		slog.Error("updating RSI info", "error", err)
		return
	}

	// memberMessage := member.GetOnboardingMessage()
	// if _, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
	// 	Channel: member.ChannelId,
	// 	ID:      member.MessageId,
	// 	Content: &memberMessage.Content,
	// 	Embeds:  &memberMessage.Embeds,
	// }); err != nil {
	// 	slog.Error("editing member message on name change", "error", err)
	// }

	nameUpdateActivity := &activity.Activity{
		Who:  member,
		When: time.Now().UTC(),
		Meta: activity.Meta{
			What: activity.NameChange,
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
