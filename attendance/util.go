package attendance

import (
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/settings"
)

func Issues(member *members.Member) []string {
	issues := []string{}

	if member.IsBot {
		issues = append(issues, "bot")
	}

	if member.IsGuest {
		issues = append(issues, "guest")
	}

	if !member.RSIMember && member.Rank != ranks.None {
		issues = append(issues, "non-rsi member but has a rank")
	}

	if !member.RSIMember && member.IsAlly {
		issues = append(issues, "marked as ally, but not a rsi member")
	}

	if member.RSIMember && member.IsAlly {
		issues = append(issues, "ally")
	}

	if member.RSIMember && member.BadAffiliation {
		issues = append(issues, "bad affiliation")
	}

	if member.RSIMember && member.PrimaryOrg == "REDACTED" {
		issues = append(issues, "redacted org")
	}

	if member.RSIMember && member.Rank <= ranks.Member && member.PrimaryOrg != settings.GetString("rsi_org_sid") {
		issues = append(issues, "bad primary org")
	}

	attendedEvents := GetMemberAttendanceCount(member.Id)
	total := member.LegacyEvents + attendedEvents

	switch member.Rank {
	case ranks.Recruit:
		if total >= 3 {
			issues = append(issues, "max event credits for this rank (3)")
		}
	case ranks.Member:
		if total >= 10 {
			issues = append(issues, "max event credits for this rank (10)")
		}
	case ranks.Technician:
		if total >= 20 {
			issues = append(issues, "max event credits for this rank (20)")
		}
	}

	return issues
}
