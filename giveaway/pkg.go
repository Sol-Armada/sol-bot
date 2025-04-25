package giveaway

import (
	"errors"

	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
)

type Giveaway struct {
	Id         string
	Items      map[string]*Item
	Attendance *attendance.Attendance
	Timer      int

	Ended     bool
	ChannelId string
	MessageId string
}

var giveaways = map[string]*Giveaway{}

func NewGiveaway(attendanceId string, items []*Item) (*Giveaway, error) {
	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return nil, err
	}

	g := &Giveaway{
		Id:         xid.New().String(),
		Items:      make(map[string]*Item),
		Attendance: attendance,
	}

	if len(items) == 0 {
		return nil, errors.New("no items")
	}

	for _, item := range items {
		if _, ok := g.Items[item.Id]; ok {
			continue
		}

		g.Items[item.Id] = item
	}

	return g, nil
}

func (g *Giveaway) SetTimer(timer int) *Giveaway {
	g.Timer = timer
	return g
}

func GetGiveaway(id string) *Giveaway {
	if g, ok := giveaways[id]; ok {
		return g
	}
	return nil
}

func (g *Giveaway) Save() *Giveaway {
	giveaways[g.Id] = g
	return g
}

func (g *Giveaway) Delete() {
	delete(giveaways, g.Id)
}

func (g *Giveaway) End() *Giveaway {
	for _, item := range g.Items {
		item.SelectWinners()
	}

	g.Ended = true
	return g.Save()
}
