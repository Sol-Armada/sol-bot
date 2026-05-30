package giveaway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/database/postgresql"
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

var giveawayPool *pgxpool.Pool

var (
	ErrGiveawayNotFound      = errors.New("giveaway not found")
	ErrUnableToUnpinGiveaway = errors.New("unable to unpin giveaway message")
)

func Setup() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	giveawayPool = pg.Pool
	return nil
}

func Load(s *discordgo.Session) error {
	if giveawayPool == nil {
		return errors.New("giveaway store not initialized")
	}

	rows, err := giveawayPool.Query(context.Background(), `
		SELECT id, name, items_json, attendance_id, end_time, ended,
			channel_id, embed_message_id, input_message_id, created_at, updated_at
		FROM giveaways
	`)
	if err != nil {
		return errors.Join(err, errors.New("failed to get giveaways from store"))
	}
	defer rows.Close()

	giveaways = make(map[string]*Giveaway)
	for rows.Next() {
		var (
			id, name, itemsJSON, channelID, embedMessageID, inputMessageID string
			attendanceID                                                   *string
			endTime                                                        *time.Time
			ended                                                          bool
			createdAt, updatedAt                                           time.Time
		)
		if err := rows.Scan(&id, &name, &itemsJSON, &attendanceID, &endTime, &ended, &channelID, &embedMessageID, &inputMessageID, &createdAt, &updatedAt); err != nil {
			return err
		}

		items := map[string]*Item{}
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
			items = map[string]*Item{}
		}

		g := &Giveaway{
			Id:             id,
			Name:           name,
			Items:          items,
			EndTime:        zeroIfNilTime(endTime),
			Ended:          ended,
			ChannelId:      channelID,
			EmbedMessageId: embedMessageID,
			InputMessageId: inputMessageID,
			sess:           s,
		}
		if attendanceID != nil {
			g.AttendanceId = *attendanceID
		}
		giveaways[g.Id] = g
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for _, g := range giveaways {
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
	if giveawayPool == nil {
		return errors.New("giveaway store not initialized")
	}

	for _, giveaway := range giveaways {
		if err := saveGiveaway(giveaway); err != nil {
			return err
		}
	}
	return nil
}

func (g *Giveaway) CanParticipate(memberId string) (bool, error) {
	if g.AttendanceId == "" {
		return true, nil
	}

	a, err := attendance.Get(g.AttendanceId)
	if err != nil {
		return false, err
	}

	return a.HasParticipant(memberId)
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
		mentions.WriteString("<@")
		mentions.WriteString(winner)
		mentions.WriteString("> ")
	}

	msg, err := g.sess.ChannelMessageSendComplex(g.ChannelId, &discordgo.MessageSend{
		Content:    mentions.String(),
		Components: g.GetComponents(),
		Embeds:     []*discordgo.MessageEmbed{g.GetEmbed()},
	})
	if err != nil {
		return errors.Join(err, errors.New("failed to send giveaway end message"))
	}

	_ = g.sess.ChannelMessagePin(msg.ChannelID, msg.ID)
	_ = g.sess.ChannelMessageDelete(g.ChannelId, g.EmbedMessageId)
	_ = g.sess.ChannelMessageDelete(g.ChannelId, g.InputMessageId)

	return SaveGiveaways()
}

func saveGiveaway(g *Giveaway) error {
	if giveawayPool == nil {
		return errors.New("giveaway store not initialized")
	}
	if g == nil {
		return nil
	}
	now := time.Now().UTC()
	if g.EndTime.IsZero() {
		g.EndTime = now
	}
	itemsJSON, err := json.Marshal(g.Items)
	if err != nil {
		return err
	}
	_, err = giveawayPool.Exec(context.Background(), `
		INSERT INTO giveaways (
			id, name, items_json, attendance_id, end_time, ended,
			channel_id, embed_message_id, input_message_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, COALESCE($10, NOW()), $11)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
			items_json = EXCLUDED.items_json,
			attendance_id = EXCLUDED.attendance_id,
			end_time = EXCLUDED.end_time,
			ended = EXCLUDED.ended,
			channel_id = EXCLUDED.channel_id,
			embed_message_id = EXCLUDED.embed_message_id,
			input_message_id = EXCLUDED.input_message_id,
			updated_at = EXCLUDED.updated_at
	`, g.Id, g.Name, string(itemsJSON), g.AttendanceId, g.EndTime.UTC(), g.Ended, g.ChannelId, g.EmbedMessageId, g.InputMessageId, now, now)
	return err
}

func zeroIfNilTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.UTC()
}
