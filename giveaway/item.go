package giveaway

import (
	"math/rand"
	"slices"

	"github.com/sol-armada/sol-bot/members"
)

type Item struct {
	Id      string            `json:"id"`
	Name    string            `json:"name"`
	Amount  int               `json:"amount"`
	Members []*members.Member `json:"members"`
}

func (i *Item) AddMember(member *members.Member) {
	if i.Members == nil {
		i.Members = []*members.Member{}
	}

	if i.HasMember(member) {
		return
	}

	i.Members = append(i.Members, member)
}

func (i *Item) RemoveMember(member *members.Member) {
	for j, m := range i.Members {
		if m.Id == member.Id {
			i.Members = slices.Delete(i.Members, j, j+1)
			return
		}
	}
}

func (i *Item) HasMember(member *members.Member) bool {
	for _, m := range i.Members {
		if m.Id == member.Id {
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

	// only keep the top N members
	for range i.Amount {
		if len(i.Members) <= i.Amount {
			break
		}

		i.Members = slices.Delete(i.Members, 0, 1)
	}
}
