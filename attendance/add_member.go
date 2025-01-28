package attendance

import "github.com/sol-armada/sol-bot/members"

func (a *Attendance) AddMember(member *members.Member) {
	defer a.removeDuplicates()

	memberIssues := Issues(member)
	if len(memberIssues) > 0 {
		a.WithIssues = append(a.WithIssues, member)
		return
	}

	a.Members = append(a.Members, member)
}
