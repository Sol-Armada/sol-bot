package giveaway

import "github.com/sol-armada/sol-bot/members"

func (g *Giveaway) GetMembersEntries(member *members.Member) []*Item {
	entries := []*Item{}

	for _, item := range g.Items {
		if item.HasMember(member) {
			entries = append(entries, item)
		}
	}

	return entries
}
