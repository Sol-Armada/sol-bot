package giveaway

import (
	"math/rand"
	"slices"
)

type Item struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Amount  int      `json:"amount"`
	Members []string `json:"members"`
}

func (i *Item) AddMember(memberId string) {
	if i.Members == nil {
		i.Members = []string{}
	}

	if i.HasMember(memberId) {
		return
	}

	i.Members = append(i.Members, memberId)
}

func (i *Item) RemoveMember(memberId string) {
	for j, mid := range i.Members {
		if mid == memberId {
			i.Members = slices.Delete(i.Members, j, j+1)
			return
		}
	}
}

func (i *Item) HasMember(memberId string) bool {
	for _, mid := range i.Members {
		if mid == memberId {
			return true
		}
	}
	return false
}

func (i *Item) SelectWinners() {
	if len(i.Members) == 0 {
		return
	}

	// shuffle the members
	for j := len(i.Members) - 1; j > 0; j-- {
		k := rand.Intn(j + 1)
		i.Members[j], i.Members[k] = i.Members[k], i.Members[j]
	}

	if len(i.Members) <= i.Amount {
		return
	}

	// only keep the top N members
	selected := []string{}
	for j := range i.Amount {
		if len(selected) > i.Amount {
			break
		}

		selected = append(selected, i.Members[j])
	}
	i.Members = selected
}
