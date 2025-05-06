package giveaway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/settings"
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
var fName = "giveaways.json"

func Load(s *discordgo.Session) error {
	fName = settings.GetString("FEATURES.GIVEAWAY.FILE")
	if fName == "" {
		fName = "giveaways.json"
	}
	b, err := os.ReadFile(fName)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("giveaways.json not found, creating new file")
			if err := os.WriteFile(fName, []byte("{}"), 0644); err != nil {
				return errors.Join(err, errors.New("failed to create giveaways.json"))
			}

			return nil
		}

		return errors.Join(err, errors.New("failed to read giveaways.json"))
	}

	if err := json.Unmarshal(b, &giveaways); err != nil {
		return errors.Join(err, errors.New("failed to unmarshal giveaways.json"))
	}

	for _, g := range giveaways {
		g.sess = s

		_ = g.UpdateMessage()
		_ = g.UpdateInputs()
	}

	go watch()

	return nil
}

func NewGiveaway(s *discordgo.Session, attendanceId string, items []*Item) (*Giveaway, error) {
	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return nil, err
	}

	g := &Giveaway{
		Id:           xid.New().String(),
		Items:        make(map[string]*Item),
		AttendanceId: attendance.Id,

		sess: s,
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

func SaveGiveaways() {
	b, err := json.MarshalIndent(giveaways, "", "  ")
	if err != nil {
		slog.Error("failed to marshal giveaways", "error", err)
		return
	}
	if err := os.WriteFile(fName, b, 0644); err != nil {
		slog.Error("failed to write giveaways.json", "error", err)
		return
	}
	slog.Debug("giveaways saved to giveaways.json")
}

func (g *Giveaway) SetEndTime(time_m int) *Giveaway {
	g.EndTime = time.Now().Add(time.Duration(time_m) * time.Minute)
	return g
}

func (g *Giveaway) Save() *Giveaway {
	giveaways[g.Id] = g
	SaveGiveaways()
	return g
}

func (g *Giveaway) Delete() {
	delete(giveaways, g.Id)

	SaveGiveaways()
}

func (g *Giveaway) End() {
	for _, item := range g.Items {
		item.SelectWinners()
	}

	if g.Ended {
		slog.Debug("giveaway already ended")
		return
	}

	g.Ended = true

	_ = g.sess.ChannelMessageUnpin(g.ChannelId, g.EmbedMessageId)

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

	mentions := ""
	for _, winner := range winners {
		mentions += "<@" + winner + "> "
	}

	msg, err := g.sess.ChannelMessageSendComplex(g.ChannelId, &discordgo.MessageSend{
		Content:    mentions,
		Components: g.GetComponents(),
		Embeds:     []*discordgo.MessageEmbed{g.GetEmbed()},
	})
	if err != nil {
		slog.Error("failed to send giveaway end message", "error", err)
	}

	_ = g.sess.ChannelMessagePin(msg.ChannelID, msg.ID)

	if err := g.sess.ChannelMessageDelete(g.ChannelId, g.EmbedMessageId); err != nil {
		slog.Error("failed to delete giveaway embeded message", "error", err)
	}

	if err := g.sess.ChannelMessageDelete(g.ChannelId, g.InputMessageId); err != nil {
		slog.Error("failed to delete giveaway input message", "error", err)
	}

	delete(giveaways, g.Id)

	SaveGiveaways()
}
