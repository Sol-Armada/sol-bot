package attendance

import (
	"fmt"
	"strings"

	"github.com/sol-armada/sol-bot/tokens"
)

func (a *Attendance) DistributeTokens() (string, error) {
	var distributedTo strings.Builder
	participants, err := a.Participants()
	if err != nil {
		return "", err
	}
	for _, participant := range participants {
		if participant.Member == nil {
			continue
		}

		member := participant.Member

		t, err := tokens.GetByMemberIdAndAttendanceId(member.Id, a.Id)
		if err != nil {
			return "", err
		}

		amount := 0
		if !hasTokensFor(t, member.Id, tokens.ReasonAttendance) {
			if err := tokens.New(member.Id, 10, tokens.ReasonAttendance, nil, &a.Id, nil).Save(); err != nil {
				return "", err
			}
			amount += 10
		}

		if a.Successful && !hasTokensFor(t, member.Id, tokens.ReasonEventSuccessful) {
			if err := tokens.New(member.Id, 20, tokens.ReasonEventSuccessful, nil, &a.Id, nil).Save(); err != nil {
				return "", err
			}
			amount += 20
		}

		if participant.StayedUntilEnd && !hasTokensFor(t, member.Id, tokens.ReasonAttendanceFull) {
			if err := tokens.New(member.Id, 10, tokens.ReasonAttendanceFull, nil, &a.Id, nil).Save(); err != nil {
				return "", err
			}
			amount += 10
		}

		if participant.IsManager && !hasTokensFor(t, member.Id, tokens.ReasonManagerBonus) && !participant.Member.IsOfficer() {
			if err := tokens.New(member.Id, 10, tokens.ReasonManagerBonus, nil, &a.Id, nil).Save(); err != nil {
				return "", err
			}
			amount += 10
		}

		if amount == 0 {
			continue
		}

		fmt.Fprintf(&distributedTo, "\n<@%s> has received %d Tokens", member.Id, amount)
	}

	return distributedTo.String(), nil
}

func hasTokensFor(tokens []tokens.TokenRecord, memberId string, reason tokens.Reason) bool {
	for _, t := range tokens {
		if t.MemberId == memberId && t.Reason == reason {
			return true
		}
	}
	return false
}
