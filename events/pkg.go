package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/kyokomi/emoji/v2"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	apierrors "github.com/sol-armada/admin/errors"
	"github.com/sol-armada/admin/events/status"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
	"go.mongodb.org/mongo-driver/bson"
)

type Repeat int

const (
	None Repeat = iota
	Daily
	Weekly
	Monthly
)

type Position struct {
	Id       string     `json:"id" bson:"id"`
	Name     string     `json:"name" bson:"name"`
	Max      int32      `json:"max" bson:"max"`
	MinRank  ranks.Rank `json:"min_rank" bson:"min_rank"`
	Members  []string   `json:"members" bson:"members"`
	Emoji    string     `json:"emoji" bson:"emoji"`
	Order    int        `json:"order" bson:"order"`
	FillLast bool       `json:"fill_last" bson:"fill_last"`
}

type Event struct {
	Id          string        `json:"_id" bson:"_id"`
	Name        string        `json:"name" bson:"name"`
	Start       time.Time     `json:"start" bson:"start"`
	End         time.Time     `json:"end" bson:"end"`
	Repeat      Repeat        `json:"repeat" bson:"repeat"`
	AutoStart   bool          `json:"auto_start" bson:"auto_start"`
	Attendees   []*user.User  `json:"attendees" bson:"attendees"`
	Status      status.Status `json:"status" bson:"status"`
	Description string        `json:"description" bson:"description"`
	Cover       string        `json:"cover" bson:"cover"`
	Positions   []*Position   `json:"positions" bson:"positions"`
	MessageId   string        `json:"message_id" bson:"message_id"`

	scheduled bool
	mu        sync.Mutex
}

var events = map[string]*Event{}

func New(body map[string]interface{}) (*Event, error) {
	name, ok := body["name"].(string)
	if !ok {
		return nil, apierrors.ErrMissingName
	}

	start, ok := body["start"].(time.Time)
	if !ok {
		return nil, apierrors.ErrMissingStart
	}

	end, ok := body["end"].(time.Time)
	if !ok {
		return nil, apierrors.ErrMissingDuration
	}

	repeatRaw, ok := body["repeat"].(float64)
	if !ok {
		repeatRaw = 0
	}
	repeat := int32(repeatRaw)

	autoStart, ok := body["auto_start"].(bool)
	if !ok {
		autoStart = false
	}

	description, ok := body["description"].(string)
	if !ok {
		description = ""
	}

	cover, ok := body["cover"].(string)
	if !ok {
		cover = ""
	}

	positionsRaw, ok := body["positions"].([]interface{})
	if !ok {
		positionsRaw = nil
	}

	positions := []*Position{}
	for _, v := range positionsRaw {
		position := &Position{}
		vJson, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(vJson, &position); err != nil {
			return nil, err
		}
		if position.Name != "" {
			positions = append(positions, position)
		}
	}

	event := &Event{
		Id:          xid.New().String(),
		Name:        name,
		Start:       start,
		End:         end,
		Repeat:      Repeat(repeat),
		Attendees:   []*user.User{},
		Status:      status.Created,
		AutoStart:   autoStart,
		Description: description,
		Cover:       cover,
		Positions:   positions,
	}

	return event, nil
}

func Get(id string) (*Event, error) {
	eventMap, err := stores.Storage.GetEvent(id)
	if err != nil {
		return nil, err
	}

	eventByte, err := json.Marshal(eventMap)
	if err != nil {
		return nil, err
	}

	event := &Event{}
	if err := json.Unmarshal(eventByte, event); err != nil {
		return nil, err
	}

	return event, nil
}

func GetAll() ([]*Event, error) {
	cur, err := stores.Storage.GetEvents(bson.D{
		{
			Key: "$and",
			Value: bson.A{
				bson.D{{Key: "end", Value: bson.D{{Key: "$gte", Value: time.Now().AddDate(0, 0, -17)}}}},
				bson.D{{Key: "status", Value: bson.D{{Key: "$lt", Value: status.Deleted}}}},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	e := []*Event{}
	if err := cur.All(context.Background(), &e); err != nil {
		return nil, err
	}

	return e, nil
}

func GetByMessageId(messageId string) (*Event, error) {
	cur, err := stores.Storage.GetEvents(bson.D{{Key: "status", Value: bson.D{{Key: "$lte", Value: status.Notified_HOUR}}}})
	if err != nil {
		return nil, err
	}

	events := []*Event{}
	if err := cur.All(context.Background(), &events); err != nil {
		return nil, err
	}

	for _, e := range events {
		if e.MessageId == messageId {
			return e, nil
		}
	}

	return nil, nil
}

func GetAllWithFilter(filter interface{}) ([]*Event, error) {
	e := []*Event{}
	cur, err := stores.Storage.GetEvents(filter)
	if err != nil {
		return nil, err
	}

	if err := cur.All(context.Background(), &e); err != nil {
		return nil, err
	}

	return e, nil
}

func getAllParticipents(e *Event) string {
	participents := ""
	pInd := 0
	for _, position := range e.Positions {
		for mInd, m := range position.Members {
			participents += "<@" + m + ">"
			if mInd < len(position.Members)-1 || pInd < len(e.Positions)-1 {
				participents += ", "
			}
		}
		pInd++
	}
	return participents
}

func EventWatcher() {
	logger := log.WithField("method", "EventWatcher")

	ticker := time.NewTicker(10 * time.Second)
	for {
		// get the events
		events, err := GetAllWithFilter(
			bson.D{{Key: "status", Value: bson.D{{Key: "$lt", Value: status.Live}}}},
		)
		if err != nil {
			logger.WithError(err).Error("getting all events")
			<-ticker.C
			continue
		}

		// look over the events
		for _, e := range events {
			logger = logger.WithField("event_id", e.Id)
		}

		<-ticker.C
	}
}

func (e *Event) Lock() {
	e.mu.Lock()
}

func (e *Event) Unlock() {
	e.mu.Unlock()
}

func (e *Event) Update(n map[string]interface{}) error {
	e.Name = n["name"].(string)
	e.Start = n["start"].(time.Time)
	e.End = n["end"].(time.Time)
	e.Description = n["description"].(string)
	e.Cover = n["cover"].(string)
	e.AutoStart = n["auto_start"].(bool)

	positionsRaw, ok := n["positions"].(map[string]interface{})
	if !ok {
		positionsRaw = nil
	}

	positions := []*Position{}
	for _, v := range positionsRaw {
		position := &Position{}
		vJson, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(vJson, &position); err != nil {
			return err
		}
		positions = append(positions, position)
	}

	e.Positions = positions

	repeatRaw, ok := n["repeat"].(float64)
	if !ok {
		repeatRaw = 0
	}

	e.Repeat = Repeat(int32(repeatRaw))

	return e.Save()
}

func (e *Event) Save() error {
	return stores.Storage.SaveEvent(e.ToMap())
}

func (e *Event) Delete() error {
	e.Status = status.Deleted
	return e.Save()
}

func (e *Event) ToMap() map[string]interface{} {
	jsonEvent, err := json.Marshal(e)
	if err != nil {
		log.WithError(err).WithField("event", e).Error("event to json")
		return map[string]interface{}{}
	}

	var mapEvent map[string]interface{}
	if err := json.Unmarshal(jsonEvent, &mapEvent); err != nil {
		log.WithError(err).WithField("event", e).Error("event to map")
		return map[string]interface{}{}
	}

	mapEvent["start"] = e.Start.UnixMilli()
	mapEvent["end"] = e.End.UnixMilli()

	return mapEvent
}

func (e *Event) Exists() bool {
	if _, err := stores.Storage.GetEvent(e.Id); err != nil {
		return false
	}

	return true
}

func (e *Event) StartEvent() error {
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting original event message")
	}

	message.Embeds[0].Fields[0].Value = fmt.Sprintf("<t:%d> - <t:%d:t>\nLive!", e.Start.Unix(), e.End.Unix())

	if _, err := b.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      message.ID,
		Channel: message.ChannelID,
		Embeds:  message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "updating original event message when live")
	}

	if err := b.MessageReactionsRemoveAll(message.ChannelID, message.ID); err != nil {
		return errors.Wrap(err, "removing all emojis")
	}

	go func() {
		timer := time.NewTimer(time.Until(e.End))
		<-timer.C

		if e.Status > status.Live {
			return
		}

		if err := e.Stop(); err != nil {
			log.WithError(err).Error("stopping event")
		}
	}()

	return nil
}

func (e *Event) Stop() error {
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting original event message")
	}

	// stop the event
	e.Status = status.Finished
	if err := e.Save(); err != nil {
		return errors.Wrap(err, "saving finished event")
	}

	message.Embeds[0].Fields[0].Value = fmt.Sprintf("<t:%d> - <t:%d:t>", e.Start.Unix(), e.End.Unix())

	if _, err := b.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      message.ID,
		Channel: message.ChannelID,
		Embeds:  message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "updating original event message when ended")
	}

	return nil
}

func (e *Event) remindParticipents(msg string) error {
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting event message")
	}

	if message.Thread == nil {
		thread, err := b.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
			Name:                "Event Notification",
			Type:                discordgo.ChannelTypeGuildPrivateThread,
			AutoArchiveDuration: 60,
		})
		if err != nil {
			return errors.Wrap(err, "reminder thread")
		}

		message.Thread = thread
	}

	participents := getAllParticipents(e)
	if participents != "" {
		if _, err := b.ChannelMessageSend(message.Thread.ID, participents+"\n\n"+msg); err != nil {
			return errors.Wrap(err, "sending reminder message")
		}
	}

	return nil
}

func (e *Event) getTimeField() string {
	timeField := fmt.Sprintf("<t:%d> - <t:%d:t>\n:timer: <t:%d:R>", e.Start.Unix(), e.End.Unix(), e.Start.Unix())
	if e.Status == status.Live {
		timeField = fmt.Sprintf("<t:%d> - <t:%d:t>\nLive!", e.Start.Unix(), e.End.Unix())
	}

	return timeField
}

func (e *Event) AllPositionsFilled() bool {
	for _, position := range e.Positions {
		if len(position.Members) < int(position.Max) {
			return false
		}
	}

	return false
}

func (e *Event) GetEmbeds() (*discordgo.MessageEmbed, error) {
	// initially add the time
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Time",
			Value: e.getTimeField(),
		},
	}

	// add the positions, in order
	emojis := emoji.CodeMap()
	// order the positions by order number
	sort.Slice(e.Positions, func(i, j int) bool {
		return e.Positions[i].Order < e.Positions[j].Order
	})
	for _, position := range e.Positions {
		name := fmt.Sprintf("%s %s (%d/%d)", emojis[strings.ToLower(position.Emoji)], position.Name, len(position.Members), position.Max)
		names := ""

		if position.Max == 0 {
			name = fmt.Sprintf("%s %s (%d/-)", emojis[strings.ToLower(position.Emoji)], position.Name, len(position.Members))
			if position.FillLast {
				names = "fill others first"
			}
			goto SKIP
		}

		for _, m := range position.Members {
			u, err := user.Get(m)
			if err != nil {
				return nil, err
			}

			// check if they have a nickname
			if u.Discord.Nick != "" {
				names += u.Discord.Nick + "\n"
				continue
			}

			// at the member to the list
			names += u.Name + "\n"
		}

		if names == "" {
			names = "-"
		}

	SKIP:
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  names,
			Inline: true,
		})
	}

	// add position rank limits
	limits := ""
	for _, position := range e.Positions {
		limits += position.Name + " - " + position.MinRank.String() + "\n"
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "Rank Limits",
		Value: limits,
	})

	// if the cover is the default logo, replace it with the link
	if e.Cover == "/logo.png" || e.Cover == "" {
		e.Cover = "https://admin.solarmada.space/logo.png"
	}

	embeds := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeArticle,
		Title:       e.Name,
		Description: e.Description,
		Fields:      fields,
		Image: &discordgo.MessageEmbedImage{
			URL: e.Cover,
		},
	}
	return embeds, nil
}

func (e *Event) NotifyOfEvent() error {
	logger := log.WithField("method", "NotifyOfEvent")

	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	embeds, err := e.GetEmbeds()
	if err != nil {
		return err
	}

	message, err := b.SendComplexMessage(config.GetString("DISCORD.CHANNELS.EVENTS"), &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			embeds,
		},
	})
	if err != nil {
		return err
	}

	if _, err := b.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
		Name:                "Event Notification",
		Type:                discordgo.ChannelTypeGuildPrivateThread,
		AutoArchiveDuration: 60,
	}); err != nil {
		return errors.Wrap(err, "starting event thread")
	}

	// make the reactions
	emojis := emoji.CodeMap()
	for _, p := range e.Positions {
		if err := b.MessageReactionAdd(message.ChannelID, message.ID, emojis[strings.ToLower(p.Emoji)]); err != nil {
			logger.WithError(err).Error("sending reaction")
		}
	}

	e.MessageId = message.ID
	e.Status = status.Announced

	t := time.Now().Add(+24 * time.Hour).UTC()
	if t.After(e.Start) {
		e.Status = status.Notified_DAY
	}

	t = time.Now().Add(+1 * time.Hour)
	if t.After(e.Start) {
		e.Status = status.Notified_HOUR
	}

	if err := e.Save(); err != nil {
		return err
	}

	return nil
}
