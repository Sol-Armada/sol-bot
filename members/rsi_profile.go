package members

import (
	rsimodule "github.com/koo04/GoScrapeRSI"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

// ResetRSIStatus clears member fields that are sourced from RSI profile data.
func (m *Member) ResetRSIStatus() {
	m.RSIMember = false
	m.IsAlly = false
	m.IsAffiliate = false
	m.IsGuest = true
	m.Rank = ranks.None
	m.PrimaryOrg = ""
	m.Affilations = []string{}
}

// ApplyRSIProfile updates member fields based on RSI profile data.
func (m *Member) ApplyRSIProfile(profile *rsimodule.UserProfile) {
	m.ResetRSIStatus()
	if profile == nil {
		return
	}

	orgSID := settings.GetString("rsi_org_sid")

	if profile.Organization.SID != "" {
		m.PrimaryOrg = profile.Organization.SID
		if profile.Organization.SID == orgSID {
			m.Rank = ranks.GetRankByRSIRankName(profile.Organization.Rank)
			m.IsGuest = false
		}
	}

	for _, aff := range profile.Affiliation {
		m.Affilations = append(m.Affilations, aff.SID)
		if aff.SID == orgSID {
			m.IsAffiliate = true
			m.Rank = ranks.Member
			m.IsGuest = false
			m.IsAlly = false
		}
	}

	m.RSIMember = true

	if utils.StringSliceContains(settings.GetStringSlice("ALLIES"), m.PrimaryOrg) {
		m.IsAlly = true
	}
}
