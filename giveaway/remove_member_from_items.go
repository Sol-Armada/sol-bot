package giveaway

func (g *Giveaway) RemoveMemberFromItems(memberId string) {
	for _, item := range g.Items {
		item.RemoveMember(memberId)
	}
	_ = g.UpdateMessage()
}
