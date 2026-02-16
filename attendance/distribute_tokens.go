package attendance

import (
	"fmt"
	"slices"

	"github.com/sol-armada/sol-bot/tokens"
)

func (a *Attendance) DistributeTokens(whoStayed []string) ([]string, error) {
	var distributedTo []string
	for _, member := range a.Members {
		t, err := tokens.GetByMemberIdAndAttendanceId(member.Id, a.Id)
		if err != nil {
			return nil, err
		}

		amount := 0
		if !hasTokensFor(t, member.Id, tokens.ReasonAttendance) {
			if err := tokens.New(member.Id, 10, tokens.ReasonAttendance, nil, &a.Id, nil).Save(); err != nil {
				return nil, err
			}
			amount += 10
		}

		if a.Successful && !hasTokensFor(t, member.Id, tokens.ReasonEventSuccessful) {
			if err := tokens.New(member.Id, 20, tokens.ReasonEventSuccessful, nil, &a.Id, nil).Save(); err != nil {
				return nil, err
			}
			amount += 20
		}

		if slices.Contains(whoStayed, member.Id) && !hasTokensFor(t, member.Id, tokens.ReasonAttendanceFull) {
			if err := tokens.New(member.Id, 10, tokens.ReasonAttendanceFull, nil, &a.Id, nil).Save(); err != nil {
				return nil, err
			}
			amount += 10
		}

		if amount == 0 {
			distributedTo = append(distributedTo, fmt.Sprintf("<@%s> already received tokens for this event", member.Id))
			continue
		}

		distributedTo = append(distributedTo, fmt.Sprintf("<@%s> has received %d Tokens", member.Id, amount))
	}

	return distributedTo, nil
}

func hasTokensFor(tokens []tokens.TokenRecord, memberId string, reason tokens.Reason) bool {
	for _, t := range tokens {
		if t.MemberId == memberId && t.Reason == reason {
			return true
		}
	}
	return false
}
