package raffles

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/mongo"
)

type Raffle struct {
	Id          string         `json:"id" bson:"_id"`
	Name        string         `json:"name"`
	AttedanceId string         `json:"attendance_id"`
	Prize       string         `json:"prize"`
	Quantity    int            `json:"quantity"`
	Tickets     map[string]int `json:"tickets"`
	Winners     []string       `json:"winners"`
	Ended       bool           `json:"ended"`
	Test        bool           `json:"test"`
	ChannelId   string         `json:"channel_id"`
	MessageId   string         `json:"message_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

var rafflesStore *stores.RaffleStore

var (
	ErrNoEntries error = errors.New("no entries in raffle")
)

func Setup() error {
	storesClient := stores.Get()
	rs, ok := storesClient.GetRafflesStore()
	if !ok {
		return errors.New("raffles store not found")
	}
	rafflesStore = rs

	return nil
}

func New(name, attendanceId, prize string, test bool) *Raffle {
	n := time.Now().UTC()

	prizeQtyStr := strings.Split(prize, ":")[1]
	prizeName := strings.Split(prize, ":")[0]
	quantity := 1
	if prizeQtyStr != "" {
		fmt.Sscanf(prizeQtyStr, "%d", &quantity)
	}

	return &Raffle{
		Id:          xid.New().String(),
		Name:        name,
		AttedanceId: attendanceId,
		Prize:       prizeName,
		Quantity:    quantity,
		Tickets:     map[string]int{},
		CreatedAt:   n,
		UpdatedAt:   n,

		Test: test,
	}
}

func Get(id string) (*Raffle, error) {
	raffle := &Raffle{}

	if err := rafflesStore.Get(id).Decode(raffle); err != nil {
		return nil, err
	}

	return raffle, nil
}

func (r *Raffle) Save() error {
	r.UpdatedAt = time.Now().UTC()
	return rafflesStore.Upsert(r.Id, r)
}

func (r *Raffle) SetMessage(message *discordgo.Message) *Raffle {
	r.ChannelId = message.ChannelID
	r.MessageId = message.ID
	return r
}

func (r *Raffle) AddTicket(memberId string, amount int) *Raffle {
	r.Tickets[memberId] = amount
	return r
}

func (r *Raffle) RemoveTicket(memberId string) *Raffle {
	delete(r.Tickets, memberId)
	return r
}

func (r *Raffle) PickWinner() ([]*members.Member, error) {
	if r.Ended {
		return nil, errors.New("raffle has ended")
	}

	// pick a winner based on weighted random
	memberIds := []string{}
	for memberId, tickets := range r.Tickets {
		for range tickets {
			memberIds = append(memberIds, memberId)
		}
	}

	if len(memberIds) == 0 {
		return nil, ErrNoEntries
	}

	winners := []*members.Member{}
	potentialWinners := slices.Clone(memberIds)
	for range r.Quantity {
		if len(potentialWinners) == 0 {
			break
		}

		selected, err := rand.Int(rand.Reader, big.NewInt(int64(len(potentialWinners))))
		if err != nil {
			return nil, err
		}

		winner, err := members.Get(potentialWinners[selected.Int64()])
		if err != nil {
			return nil, err
		}

		winners = append(winners, winner)
		r.Winners = append(r.Winners, winner.Id)

		// remove all entries of the winner from potentialWinners
		newPotentialWinners := slices.DeleteFunc(potentialWinners, func(id string) bool {
			return id == winner.Id
		})
		potentialWinners = newPotentialWinners
	}
	r.Ended = true

	return winners, r.Save()
}

func (r *Raffle) GetTickets() string {
	if len(r.Tickets) == 0 {
		return "No tickets"
	}

	sorted := make([]struct {
		MemberId string
		Amount   int
	}, 0, len(r.Tickets))

	for memberId, amount := range r.Tickets {
		sorted = append(sorted, struct {
			MemberId string
			Amount   int
		}{memberId, amount})
	}

	slices.SortFunc(sorted, func(a, b struct {
		MemberId string
		Amount   int
	}) int {
		return a.Amount - b.Amount
	})

	tickets := ""
	for _, ticket := range sorted {
		tickets += fmt.Sprintf("<@%s>", ticket.MemberId)
		if len(r.Winners) != 0 {
			tickets += fmt.Sprintf(" | %d", ticket.Amount)
		}
		tickets += "\n"
	}
	return tickets
}

func (r *Raffle) GetLatest() (*Raffle, error) {
	latestRaffle := &Raffle{}
	cur, err := rafflesStore.GetLatest()
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	if !cur.Next(context.TODO()) {
		return nil, nil
	}

	if err := cur.Decode(latestRaffle); err != nil {
		return nil, err
	}

	return latestRaffle, nil
}

func (r *Raffle) MemberWonLast(id string) (bool, error) {
	latestRaffle, err := r.GetLatest()
	if err != nil {
		return false, err
	}

	if latestRaffle == nil {
		return false, nil
	}

	if len(latestRaffle.Winners) == 0 {
		return false, nil
	}

	for _, winnerId := range latestRaffle.Winners {
		if winnerId == id {
			return true, nil
		}
	}

	return false, nil
}
func (r *Raffle) Delete() error {
	return rafflesStore.Delete(r.Id)
}
