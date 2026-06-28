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
	logger := slog.Default().With(
		slog.String("event", "name_change"),
	)

	logger.Debug("member name changed", "user_id", func() string {
		if m.User != nil {
			return m.User.ID
		}
		return "user is nil"
	}, "old_name", func() string {
		if m.Member != nil {
			return m.Member.Nick
		}
		return "member is nil"
	}(), "new_name", func() string {
		if m.BeforeUpdate != nil {
			return m.BeforeUpdate.Nick
		}
		return "before update is nil"
	}())

	if m.User != nil && m.User.Bot {
		return
	}

	if m.User == nil {
		logger.Error("member update event missing user")
		return
	}

	member, err := members.Get(m.User.ID)
	if err != nil {
		logger.Error("getting member", "error", err)
		return
	}

	member.Name = m.User.Username
	if err := member.Save(); err != nil {
		logger.Error("saving member", "error", err)
		return
	}

	member.UpdateRank(m.Member.Roles)

	ebo := utils.NewExponentialBackoff(1, time.Second, 1.1, 3, logger)

	if err := ebo.Execute(func() error {
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
		logger.Error("updating RSI info", "error", err)
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
		logger.Error("saving name change activity", "error", err)
	}
}
