package attendance

import (
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
)

func Issues(m *members.Member) []string {
	issues := []string{}

	if m.IsBot {
		issues = append(issues, "bot")
	}

	if m.Rank == ranks.Guest {
		issues = append(issues, "guest")
	}

	if m.Rank == ranks.Recruit && !m.RSIMember {
		issues = append(issues, "non-rsi member but is recruit")
	}

	if m.IsAlly {
		issues = append(issues, "ally")
	}

	if m.BadAffiliation {
		issues = append(issues, "bad affiliation")
	}

	if m.PrimaryOrg == "REDACTED" {
		issues = append(issues, "redacted org")
	}

	if m.Rank <= ranks.Member && m.PrimaryOrg != config.GetString("rsi_org_sid") {
		issues = append(issues, "bad primary org")
	}

	switch m.Rank {
	case ranks.Recruit:
		if m.Events >= 3 {
			issues = append(issues, "max event credits for this rank (3)")
		}
	case ranks.Member:
		if m.Events >= 10 {
			issues = append(issues, "max event credits for this rank (10)")
		}
	case ranks.Technician:
		if m.Events >= 20 {
			issues = append(issues, "max event credits for this rank (20)")
		}
	}

	return issues
}
