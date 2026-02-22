package attendance

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/utils"
)

func (a *Attendance) RecheckIssues(s *discordgo.Session) error {
	attendees := []*members.Member{}
	for _, member := range a.Members {
		issues := Issues(member)

		if len(issues) == 0 {
			attendees = append(attendees, member)
		} else {
			a.WithIssues = append(a.WithIssues, member)
		}
	}
	a.Members = attendees

	newIssues := []*members.Member{}
	for _, member := range a.WithIssues {
		dm, err := s.GuildMember(s.State.Application.GuildID, member.Id)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("getting guild member for <@%s>", member.Id))
			// slog.Default().Warn("getting guild member", "member", fmt.Sprintf("<@%s>", member.Id), "error", err)
			// goto SKIP
		}

		member.UpdateRoles(dm.Roles)

		if err := rsi.UpdateMemberRSIInfo(member, &utils.ExponentialBackoff{
			MaxRetries: 3,
			Multiplier: 1.1,
			MaxDelay:   time.Second,
		}, slog.Default()); err != nil {
			return errors.Wrap(err, fmt.Sprintf("updating RSI info for <@%s>", member.Id))
		}

		// SKIP:
		memberIssues := Issues(member)
		if len(memberIssues) != 0 {
			newIssues = append(newIssues, member)
		} else {
			a.Members = append(a.Members, member)
		}
	}
	a.WithIssues = newIssues

	a.removeDuplicates()

	return a.Save()
}
