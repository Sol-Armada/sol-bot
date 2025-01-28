package sos

import "context"

func GetTicketsByMemberId(id string) []*Ticket {
	cur, err := store.GetSOSTicketsByMemberId(id)
	if err != nil {
		return nil
	}

	tickets := []*Ticket{}

	for cur.Next(context.Background()) {
		ticket := Ticket{}
		if err := cur.Decode(&ticket); err != nil {
			return nil
		}
		tickets = append(tickets, &ticket)
	}

	return tickets
}
