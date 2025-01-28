package attendance

import "github.com/sol-armada/sol-bot/members"

func (a *Attendance) RemoveMember(member *members.Member) {
	for i, m := range a.Members {
		if m.Id == member.Id {
			a.Members = append(a.Members[:i], a.Members[i+1:]...)
			break
		}
	}

	for i, m := range a.WithIssues {
		if m.Id == member.Id {
			a.WithIssues = append(a.WithIssues[:i], a.WithIssues[i+1:]...)
			break
		}
	}

	a.removeDuplicates()
}
