package giveaway

import (
	"slices"
)

func (g *Giveaway) AddMemberToItems(items []string, memberId string) *Giveaway {
	for _, itemId := range items {
		item, ok := g.Items[itemId]
		if !ok {
			return g
		}

		item.AddMember(memberId)
	}

	for _, item := range g.Items {
		if !slices.Contains(items, item.Id) {
			item.RemoveMember(memberId)
		}
	}

	return g.Save()
}
