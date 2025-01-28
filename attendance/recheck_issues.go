package attendance

import "github.com/sol-armada/sol-bot/members"

func (a *Attendance) RecheckIssues() error {
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
