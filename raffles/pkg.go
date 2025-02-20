package raffles

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/stores"
)

type Raffle struct {
	Id          string         `json:"id" bson:"_id"`
	AttedanceId string         `json:"attendance_id"`
	Prize       string         `json:"prize"`
	Tickets     map[string]int `json:"tickets"`
	WinnerId    string         `json:"winner_id"`
	Ended       bool           `json:"ended"`
	ChannelId   string         `json:"channel_id"`
	MessageId   string         `json:"message_id"`
}

var rafflesStore *stores.RaffleStore

func Setup() error {
	storesClient := stores.Get()
	rs, ok := storesClient.GetRafflesStore()
	if !ok {
		return errors.New("raffles store not found")
	}
	rafflesStore = rs

	return nil
}

func New(attendanceId, prize string) *Raffle {
	return &Raffle{
		Id:          xid.New().String(),
		AttedanceId: attendanceId,
		Prize:       prize,
		Tickets:     map[string]int{},
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

func (r *Raffle) PickWinner() (*members.Member, error) {
	if r.Ended {
		return nil, errors.New("raffle has ended")
	}

	// pick a winner based on weighted random
	memberIds := []string{}
	for memberId, tickets := range r.Tickets {
		for i := 0; i < tickets; i++ {
			memberIds = append(memberIds, memberId)
		}
	}

	if len(memberIds) == 0 {
		return nil, errors.New("no tickets")
	}

	selected, err := rand.Int(rand.Reader, big.NewInt(int64(len(memberIds))))
	if err != nil {
		return nil, err
	}

	winner, err := members.Get(memberIds[selected.Int64()])
	if err != nil {
		return nil, err
	}

	r.WinnerId = winner.Id
	r.Ended = true

	return winner, r.Save()
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
		tickets += fmt.Sprintf("<@%s>: %d\n", ticket.MemberId, ticket.Amount)
	}
	return tickets
}

func (r *Raffle) GetEmbed() (*discordgo.MessageEmbed, error) {
	attendanceRecord, err := attendance.Get(r.AttedanceId)
	if err != nil {
		return nil, err
	}

	feilds := []*discordgo.MessageEmbedField{
		{
			Name:   "Event",
			Value:  attendanceRecord.Name,
			Inline: true,
		},
		{
			Name:   "Prize",
			Value:  r.Prize,
			Inline: true,
		},
		{
			Name:  "Tickets",
			Value: r.GetTickets(),
		},
	}

	if r.Ended {
		winner, err := members.Get(r.WinnerId)
		if err != nil {
			return nil, err
		}

		feilds = append(feilds, &discordgo.MessageEmbedField{
			Name:   "🥳 Winner 🎉",
			Value:  fmt.Sprintf("<@%s>", winner.Id),
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:  "Raffle",
		Fields: feilds,
	}, nil
}

func (r *Raffle) UpdateMessage(s *discordgo.Session) error {
	embed, err := r.GetEmbed()
	if err != nil {
		return err
	}

	_, err = s.ChannelMessageEditEmbed(r.ChannelId, r.MessageId, embed)
	return err
}
