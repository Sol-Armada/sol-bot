package attendance

import (
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
)

func Issues(member *members.Member) []string {
	issues := []string{}

	if member.IsBot {
		issues = append(issues, "bot")
		return issues
	}

	// if member.OnboardedAt == nil {
	// 	issues = append(issues, "not onboarded")
	// 	return issues
	// }

	if member.IsGuest {
		issues = append(issues, "guest")
		return issues
	}

	if !member.RSIMember {
		issues = append(issues, "not on rsi")
		return issues
	}

	if !member.RSIMember && member.IsAlly {
		issues = append(issues, "marked as ally, but not a rsi member")
		return issues
	}

	if member.RSIMember && member.IsAffiliate {
		issues = append(issues, "is affiliate")
		return issues
	}

	if member.RSIMember && member.BadAffiliation {
		issues = append(issues, "bad affiliation")
	}

	if member.RSIMember && member.PrimaryOrg == "REDACTED" {
		issues = append(issues, "redacted org")
		return issues
	}

	if member.RSIMember && member.Rank <= ranks.Technician && member.PrimaryOrg != settings.GetString("rsi_org_sid") {
		issues = append(issues, "ranked, but org not set as primary")
		return issues
	}

	if member.RSIMember && member.IsAlly {
		issues = []string{}
	}

	// attendedEvents, err := GetMemberAttendanceCount(member.Id)
	// if err != nil {
	// 	return issues
	// }

	// switch member.Rank {
	// case ranks.Recruit:
	// 	if attendedEvents >= 3 {
	// 		issues = append(issues, "max event credits for this rank (3)")
	// 	}
	// case ranks.Member:
	// 	if attendedEvents >= 10 {
	// 		issues = append(issues, "max event credits for this rank (10)")
	// 	}
	// case ranks.Technician:
	// 	if attendedEvents >= 20 {
	// 		issues = append(issues, "max event credits for this rank (20)")
	// 	}
	// }

	return issues
}
