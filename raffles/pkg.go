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

	"github.com/sol-armada/sol-bot/members"
)

type Raffle struct {
	Id           int32          `json:"id"`
	Name         string         `json:"name"`
	AttendanceId string         `json:"attendance_id"`
	Prize        string         `json:"prize"`
	Quantity     int32          `json:"quantity"`
	Tickets      map[string]int `json:"tickets"`
	Winners      []string       `json:"winners"`
	Ended        bool           `json:"ended"`
	Test         bool           `json:"test"`
	ChannelId    string         `json:"channel_id"`
	MessageId    string         `json:"message_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

var (
	ErrNoEntries error = errors.New("no entries in raffle")
)

var rafflesBackend raffleBackend

func Setup() error {
	return setupRafflesBackend()
}

func New(name, attendanceId, prize string, test bool) (*Raffle, error) {
	n := time.Now().UTC()

	prizeSplit := strings.Split(prize, ":")
	prizeName := strings.Split(prize, ":")[0]
	var prizeQtyStr string
	if len(prizeSplit) > 1 {
		prizeQtyStr = strings.Split(prize, ":")[1]
	}

	quantity := int32(1)
	if prizeQtyStr != "" {
		_, _ = fmt.Sscanf(prizeQtyStr, "%d", &quantity)
	}

	newRaffle := &Raffle{
		Name:         name,
		AttendanceId: attendanceId,
		Prize:        prizeName,
		Quantity:     quantity,
		Tickets:      map[string]int{},
		CreatedAt:    n,
		UpdatedAt:    n,

		Test: test,
	}

	if rafflesBackend == nil {
		return nil, errors.New("raffles backend not initialized")
	}

	id, err := rafflesBackend.Insert(context.Background(), newRaffle)
	if err != nil {
		return nil, err
	}
	newRaffle.Id = id

	return newRaffle, nil
}

func Get(id int32) (*Raffle, error) {
	if rafflesBackend == nil {
		return nil, errors.New("raffles backend not initialized")
	}

	return rafflesBackend.Get(context.Background(), id)
}

func (r *Raffle) Save(ctx context.Context) (err error) {
	if rafflesBackend == nil {
		return errors.New("raffles backend not initialized")
	}

	if err = rafflesBackend.Update(ctx, r); err != nil {
		return err
	}

	return nil
}

func (r *Raffle) AddTicket(memberId string, amount int) *Raffle {
	r.Tickets[memberId] = amount
	return r
}

func (r *Raffle) RemoveTicket(memberId string) *Raffle {
	delete(r.Tickets, memberId)
	return r
}

func (r *Raffle) PickWinner(ctx context.Context) ([]*members.Member, error) {
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

	return winners, r.Save(ctx)
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

	var tickets strings.Builder
	for _, ticket := range sorted {
		fmt.Fprintf(&tickets, "<@%s>", ticket.MemberId)
		if len(r.Winners) != 0 {
			fmt.Fprintf(&tickets, " | %d", ticket.Amount)
		}
		tickets.WriteString("\n")
	}
	return tickets.String()
}

func (r *Raffle) GetLatest(ctx context.Context) (*Raffle, error) {
	if rafflesBackend == nil {
		return nil, errors.New("raffles backend not initialized")
	}

	return rafflesBackend.GetLatest(ctx)
}

func (r *Raffle) MemberWonLast(ctx context.Context, id string) (bool, error) {
	latestRaffle, err := r.GetLatest(ctx)
	if err != nil {
		return false, err
	}

	if latestRaffle == nil {
		return false, nil
	}

	if len(latestRaffle.Winners) == 0 {
		return false, nil
	}

	if slices.Contains(latestRaffle.Winners, id) {
		return true, nil
	}

	return false, nil
}

func (r *Raffle) Delete() error {
	if rafflesBackend == nil {
		return errors.New("raffles backend not initialized")
	}

	return rafflesBackend.Delete(context.Background(), r.Id)
}
