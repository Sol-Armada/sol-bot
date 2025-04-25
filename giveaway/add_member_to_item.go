package giveaway

import (
	"slices"

	"github.com/sol-armada/sol-bot/members"
)

func (g *Giveaway) AddMemberToItems(items []string, member *members.Member) *Giveaway {
	for _, itemId := range items {
		item, ok := g.Items[itemId]
		if !ok {
			return g
		}

		item.AddMember(member)
	}

	for _, item := range g.Items {
		if !slices.Contains(items, item.Id) {
			item.RemoveMember(member)
		}
	}

	return g.Save()
}
