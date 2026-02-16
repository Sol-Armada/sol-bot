package giveaway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/stores"
)

type Giveaway struct {
	Id           string           `json:"id"`
	Name         string           `json:"name"`
	Items        map[string]*Item `json:"items"`
	AttendanceId string           `json:"attendance_id"`
	EndTime      time.Time        `json:"end_time"`

	Ended          bool   `json:"ended"`
	ChannelId      string `json:"channel_id"`
	EmbedMessageId string `json:"embed_message_id"`
	InputMessageId string `json:"input_message_id"`

	sess *discordgo.Session `json:""`
}

var giveaways = map[string]*Giveaway{}

var giveawayStore *stores.GiveawaysStore

func Setup() error {
	storesClient := stores.Get()
	gs, ok := storesClient.GetGiveawaysStore()
	if !ok {
		return errors.New("failed to get giveaways store")
	}
	giveawayStore = gs
	return nil
}

func Load(s *discordgo.Session) error {
	cur, err := giveawayStore.GetAll()
	if err != nil {
		return errors.Join(err, errors.New("failed to get giveaways from store"))
	}
	defer cur.Close(context.Background())

	var gList []*Giveaway
	if err := cur.All(context.Background(), &gList); err != nil {
		return errors.Join(err, errors.New("failed to decode giveaways from store"))
	}

	giveaways = make(map[string]*Giveaway)
	for _, g := range gList {
		giveaways[g.Id] = g
	}

	for _, g := range giveaways {
		g.sess = s

		if err := g.UpdateMessage(); err != nil {
			slog.Error("failed to update giveaway message", "error", err)
		}
		if err := g.UpdateInputs(); err != nil {
			slog.Error("failed to update giveaway inputs", "error", err)
		}
	}

	go watch()

	return nil
}

func NewGiveaway(s *discordgo.Session, name, attendanceId string, items []*Item) (*Giveaway, error) {
	g := &Giveaway{
		Id:    xid.New().String(),
		Name:  name,
		Items: make(map[string]*Item),
		sess:  s,
	}

	var a *attendance.Attendance
	if attendanceId != "" {
		var err error
		a, err = attendance.Get(attendanceId)
		if err != nil {
			return nil, err
		}
	}

	if a != nil {
		g.AttendanceId = a.Id
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

func GetGiveaway(id string) *Giveaway {
	if g, ok := giveaways[id]; ok {
		return g
	}
	return nil
}

func SaveGiveaways() error {
	giveawaysAny := make(map[string]any)
	for id, giveaway := range giveaways {
		giveawaysAny[id] = giveaway
	}

	return giveawayStore.UpsertAll(giveawaysAny)
}

func (g *Giveaway) CanParticipate(memberId string) bool {
	if g.AttendanceId == "" {
		return true
	}

	a, err := attendance.Get(g.AttendanceId)
	if err != nil {
		slog.Error("failed to get attendance for giveaway", "error", err)
		return false
	}

	return a.HasMember(memberId, true)
}

func (g *Giveaway) SetEndTime(time_m int) *Giveaway {
	g.EndTime = time.Now().Add(time.Duration(time_m) * time.Minute)
	return g
}

func (g *Giveaway) Save() (*Giveaway, error) {
	giveaways[g.Id] = g
	return g, SaveGiveaways()
}

func (g *Giveaway) End() error {
	if g.Ended {
		slog.Debug("giveaway already ended")
		return nil
	}

	for _, item := range g.Items {
		item.SelectWinners()
	}

	g.Ended = true

	if err := g.sess.ChannelMessageUnpin(g.ChannelId, g.EmbedMessageId); err != nil {
		return errors.Join(err, errors.New("failed to unpin giveaway embed message"))
	}

	// get the winners
	winners := []string{}
	for _, item := range g.Items {
		if item.Members == nil {
			continue
		}

		if len(item.Members) == 0 {
			continue
		}

		winners = append(winners, item.Members...)
	}

	// remove duplicates
	winnersMap := map[string]bool{}
	for _, winner := range winners {
		winnersMap[winner] = true
	}
	winners = []string{}
	for winner := range winnersMap {
		winners = append(winners, winner)
	}

	var mentions strings.Builder
	for _, winner := range winners {
		mentions.WriteString("<@" + winner + "> ")
	}

	msg, err := g.sess.ChannelMessageSendComplex(g.ChannelId, &discordgo.MessageSend{
		Content:    mentions.String(),
		Components: g.GetComponents(),
		Embeds:     []*discordgo.MessageEmbed{g.GetEmbed()},
	})
	if err != nil {
		return errors.Join(err, errors.New("failed to send giveaway end message"))
	}

	if err := g.sess.ChannelMessagePin(msg.ChannelID, msg.ID); err != nil {
		return errors.Join(err, errors.New("failed to pin giveaway end message"))
	}

	if err := g.sess.ChannelMessageDelete(g.ChannelId, g.EmbedMessageId); err != nil {
		return errors.Join(err, errors.New("failed to delete giveaway embeded message"))
	}

	if err := g.sess.ChannelMessageDelete(g.ChannelId, g.InputMessageId); err != nil {
		return errors.Join(err, errors.New("failed to delete giveaway input message"))
	}

	return SaveGiveaways()
}
