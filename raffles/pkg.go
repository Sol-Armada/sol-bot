package raffles

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/members"
)

type Raffle struct {
	Id          string         `json:"id" `
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

var rafflesPool *pgxpool.Pool

var (
	ErrNoEntries error = errors.New("no entries in raffle")
)

func Setup() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	rafflesPool = pg.Pool

	return nil
}

func New(name, attendanceId, prize string, test bool) *Raffle {
	n := time.Now().UTC()

	prizeSplit := strings.Split(prize, ":")
	prizeName := strings.Split(prize, ":")[0]
	var prizeQtyStr string
	if len(prizeSplit) > 1 {
		prizeQtyStr = strings.Split(prize, ":")[1]
	}

	quantity := 1
	if prizeQtyStr != "" {
		_, _ = fmt.Sscanf(prizeQtyStr, "%d", &quantity)
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
	if rafflesPool == nil {
		return nil, errors.New("raffles store not initialized")
	}

	var (
		raffle       Raffle
		ticketsJSON  string
		attendanceID *string
	)
	err := rafflesPool.QueryRow(context.Background(), `
		SELECT id, name, attendance_id, prize, quantity, tickets_json, winners,
			ended, test, channel_id, message_id, created_at, updated_at
		FROM raffles
		WHERE id = $1
	`, id).Scan(
		&raffle.Id,
		&raffle.Name,
		&attendanceID,
		&raffle.Prize,
		&raffle.Quantity,
		&ticketsJSON,
		&raffle.Winners,
		&raffle.Ended,
		&raffle.Test,
		&raffle.ChannelId,
		&raffle.MessageId,
		&raffle.CreatedAt,
		&raffle.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if attendanceID != nil {
		raffle.AttedanceId = *attendanceID
	}
	if err := json.Unmarshal([]byte(ticketsJSON), &raffle.Tickets); err != nil {
		raffle.Tickets = map[string]int{}
	}
	if raffle.Tickets == nil {
		raffle.Tickets = map[string]int{}
	}

	return &raffle, nil
}

func (r *Raffle) Save() error {
	if rafflesPool == nil {
		return errors.New("raffles store not initialized")
	}
	r.UpdatedAt = time.Now().UTC()
	ticketsJSON, err := json.Marshal(r.Tickets)
	if err != nil {
		return err
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = r.UpdatedAt
	}
	_, err = rafflesPool.Exec(context.Background(), `
		INSERT INTO raffles (
			id, name, attendance_id, prize, quantity, tickets_json, winners,
			ended, test, channel_id, message_id, created_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
			attendance_id = EXCLUDED.attendance_id,
			prize = EXCLUDED.prize,
			quantity = EXCLUDED.quantity,
			tickets_json = EXCLUDED.tickets_json,
			winners = EXCLUDED.winners,
			ended = EXCLUDED.ended,
			test = EXCLUDED.test,
			channel_id = EXCLUDED.channel_id,
			message_id = EXCLUDED.message_id,
			updated_at = EXCLUDED.updated_at
	`, r.Id, r.Name, r.AttedanceId, r.Prize, r.Quantity, string(ticketsJSON), r.Winners, r.Ended, r.Test, r.ChannelId, r.MessageId, r.CreatedAt.UTC(), r.UpdatedAt.UTC())
	return err
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

	var tickets strings.Builder
	for _, ticket := range sorted {
		tickets.WriteString(fmt.Sprintf("<@%s>", ticket.MemberId))
		if len(r.Winners) != 0 {
			tickets.WriteString(fmt.Sprintf(" | %d", ticket.Amount))
		}
		tickets.WriteString("\n")
	}
	return tickets.String()
}

func (r *Raffle) GetLatest() (*Raffle, error) {
	if rafflesPool == nil {
		return nil, errors.New("raffles store not initialized")
	}

	var (
		latestRaffle Raffle
		ticketsJSON  string
		attendanceID *string
	)
	err := rafflesPool.QueryRow(context.TODO(), `
		SELECT id, name, attendance_id, prize, quantity, tickets_json, winners,
			ended, test, channel_id, message_id, created_at, updated_at
		FROM raffles
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(
		&latestRaffle.Id,
		&latestRaffle.Name,
		&attendanceID,
		&latestRaffle.Prize,
		&latestRaffle.Quantity,
		&ticketsJSON,
		&latestRaffle.Winners,
		&latestRaffle.Ended,
		&latestRaffle.Test,
		&latestRaffle.ChannelId,
		&latestRaffle.MessageId,
		&latestRaffle.CreatedAt,
		&latestRaffle.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if attendanceID != nil {
		latestRaffle.AttedanceId = *attendanceID
	}
	if err := json.Unmarshal([]byte(ticketsJSON), &latestRaffle.Tickets); err != nil {
		latestRaffle.Tickets = map[string]int{}
	}

	return &latestRaffle, nil
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

	if slices.Contains(latestRaffle.Winners, id) {
		return true, nil
	}

	return false, nil
}
func (r *Raffle) Delete() error {
	if rafflesPool == nil {
		return errors.New("raffles store not initialized")
	}
	_, err := rafflesPool.Exec(context.Background(), `DELETE FROM raffles WHERE id = $1`, r.Id)
	return err
}
