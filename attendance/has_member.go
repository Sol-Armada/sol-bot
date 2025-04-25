package attendance

func (a *Attendance) HasMember(memberId string, includeIssues bool) bool {
	for _, member := range a.Members {
		if member.Id == memberId {
			return true
		}
	}

	if includeIssues {
		for _, issue := range a.WithIssues {
			if issue.Id == memberId {
				return true
			}
		}
	}

	return false
}
