package attendance

import (
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
)

func Issues(member *members.Member) []string {
	issues := []string{}

	if member == nil {
		return issues
	}

	if member.IsBot {
		issues = append(issues, "bot")
		return issues
	}

	if member.Rank == ranks.None || member.Rank == ranks.Guest {
		issues = append(issues, "guest")
		return issues
	}

	if !member.OnRsi() {
		issues = append(issues, "not on rsi")
		return issues
	}

	if !member.OnRsi() && member.IsAlly {
		issues = append(issues, "marked as ally, but not a rsi member")
		return issues
	}

	if member.OnRsi() && member.IsAffiliate {
		issues = append(issues, "is affiliate")
		return issues
	}

	if member.OnRsi() && member.BadAffiliation() {
		issues = append(issues, "bad affiliation")
	}

	if member.OnRsi() && member.RsiInfo.PrimaryOrgSid == "REDACTED" {
		issues = append(issues, "redacted org")
		return issues
	}

	if member.OnRsi() && member.Rank <= ranks.Technician && member.RsiInfo.PrimaryOrgSid != "SOLARMADA" {
		issues = append(issues, "ranked, but org not set as primary")
		return issues
	}

	if member.OnRsi() && member.IsAlly {
		issues = []string{}
	}

	return issues
}
