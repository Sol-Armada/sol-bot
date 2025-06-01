package attendance

import "github.com/sol-armada/sol-bot/members"

func (a *Attendance) GetMembers(withIssues bool) []*members.Member {
	membersList := a.Members
	if withIssues {
		membersList = append(membersList, a.WithIssues...)
	}
	return membersList
}
