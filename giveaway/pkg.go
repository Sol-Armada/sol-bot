package giveaway

import (
	"errors"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
)

type Giveaway struct {
	Id             string
	Name           string
	Items          map[string]*Item
	Attendance     *attendance.Attendance
	TimeRemainingS int

	Ended          bool
	ChannelId      string
	EmbedMessageId string
	InputMessageId string
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

func (g *Giveaway) SetTimer(time_m int) *Giveaway {
	g.TimeRemainingS = time_m * 60
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
	delete(giveaways, g.Id)
	return g.Save()
}

func (g *Giveaway) StartTimer(s *discordgo.Session) {
	go func() {

		for g.TimeRemainingS >= 0 {
			if g.Ended {
				return
			}

			g.TimeRemainingS--
			if err := g.UpdateMessage(s); err != nil {
				slog.Error("failed to update giveaway message", "error", err)
			}
			time.Sleep(1 * time.Second)
		}

		g.End()

		_ = s.ChannelMessageUnpin(g.ChannelId, g.EmbedMessageId)

		if _, err := s.ChannelMessageSendComplex(g.ChannelId, &discordgo.MessageSend{
			Components: g.GetComponents(),
			Embeds:     []*discordgo.MessageEmbed{g.GetEmbed()},
		}); err != nil {
			slog.Error("failed to send giveaway end message", "error", err)
		}

		if err := s.ChannelMessageDelete(g.ChannelId, g.EmbedMessageId); err != nil {
			slog.Error("failed to delete giveaway embeded message", "error", err)
		}

		if err := s.ChannelMessageDelete(g.ChannelId, g.InputMessageId); err != nil {
			slog.Error("failed to delete giveaway input message", "error", err)
		}
	}()
}
