package giveaway

func (g *Giveaway) GetMembersEntries(memberId string) []*Item {
	entries := []*Item{}

	for _, item := range g.Items {
		if item.HasMember(memberId) {
			entries = append(entries, item)
		}
	}

	return entries
}
