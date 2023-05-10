package events

import (
	"context"
	"encoding/json"
	"fmt"
	"os/user"
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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repeat int

const (
	None Repeat = iota
	Daily
	Weekly
	Monthly
)

type Position struct {
	Id      string     `json:"id" bson:"id"`
	Name    string     `json:"name" bson:"name"`
	Max     int32      `json:"max" bson:"max"`
	MinRank ranks.Rank `json:"min_rank" bson:"min_rank"`
	Members []string   `json:"members" bson:"members"`
	Emoji   string     `json:"emoji" bson:"emoji"`
	Order   int        `json:"order" bson:"order"`
}

type Event struct {
	Id          string               `json:"_id" bson:"_id"`
	Name        string               `json:"name" bson:"name"`
	Start       time.Time            `json:"start" bson:"start"`
	End         time.Time            `json:"end" bson:"end"`
	Repeat      Repeat               `json:"repeat" bson:"repeat"`
	AutoStart   bool                 `json:"auto_start" bson:"auto_start"`
	Attendees   []*user.User         `json:"attendees" bson:"attendees"`
	Status      status.Status        `json:"status" bson:"status"`
	Description string               `json:"description" bson:"description"`
	Cover       string               `json:"cover" bson:"cover"`
	Positions   map[string]*Position `json:"positions" bson:"positions"`
	MessageId   string               `json:"message_id" bson:"message_id"`

	mu sync.Mutex
}

var scheduled = map[string]*Event{}

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

	positionsRaw, ok := body["positions"].(map[string]interface{})
	if !ok {
		positionsRaw = nil
	}

	positions := map[string]*Position{}
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
			positions[position.Id] = position
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
	cur, err := stores.Storage.GetEvents(bson.D{{Key: "end", Value: bson.D{{Key: "$gte", Value: time.Now().AddDate(0, 0, -17)}}}})
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

	positions := map[string]*Position{}
	for _, v := range positionsRaw {
		position := &Position{}
		vJson, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(vJson, &position); err != nil {
			return err
		}
		positions[position.Id] = position
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
	delete(scheduled, e.Id)
	return stores.Storage.DeleteEvent(e.ToMap())
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

// Schedule ...
func (e *Event) Schedule() {
	// Remind a day before
	go e.reminderOfEventDay()

	// Remind an hour before
	go e.reminderOfEventHour()

	// Alert that the event has started
	timer := time.NewTimer(time.Until(e.Start))
	<-timer.C

	if _, ok := scheduled[e.Id]; ok {
		if err := e.StartEvent(); err != nil {
			log.WithError(err).Error("starting event")
		}
	}
}

func (e *Event) StartEvent() error {
	logger := log.WithField("event start", e)
	logger.Info("starting event")

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

	// alert the event is live
	participents := getAllParticipents(e)
	logger.WithField("participents", participents).Debug("participents")
	if participents != "" {
		if _, err := b.ChannelMessageSend(message.Thread.ID, participents+"\n\nEvent is live!"); err != nil {
			return errors.Wrap(err, "sending event starting message")
		}
	}

	// mark event as live
	e.Status = status.Live
	if err := e.Save(); err != nil {
		return errors.Wrap(err, "saving finished event")
	}

	timer := time.NewTimer(time.Until(e.End))
	<-timer.C

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

func (e *Event) reminderOfEventDay() {
	logger := log.WithField("method", "ReminderOfEventDay")

	if e.Status >= status.Notified_DAY {
		return
	}

	b, err := bot.GetBot()
	if err != nil {
		logger.WithError(err).Error("getting bot")
		return
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		logger.WithError(err).Error("getting original event message")
		return
	}

	timer := time.NewTimer(time.Until(e.Start.Add(-24 * time.Hour)))
	<-timer.C

	if _, ok := scheduled[e.Id]; ok {
		if e.Status != status.Live {
			if message.Thread == nil {
				thread, err := b.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
					Name:                "Event Notification",
					Type:                discordgo.ChannelTypeGuildPrivateThread,
					AutoArchiveDuration: 60,
				})
				if err != nil {
					logger.WithError(err).Error("starting event thread")
					return
				}

				message.Thread = thread
			}

			e.Status = status.Notified_DAY
			if err := e.Save(); err != nil {
				logger.WithError(err).Error("saving notification day event")
				return
			}

			// alert the event is live
			participents := getAllParticipents(e)
			if participents != "" {
				if _, err := b.ChannelMessageSend(message.Thread.ID, participents+"\n\nReminder that this event is happening tomorrow!"); err != nil {
					logger.WithError(err).Error("sending event starting message")
					return
				}
			}
		}
	}
}

func (e *Event) reminderOfEventHour() {
	logger := log.WithField("method", "ReminderOfEventHour")

	if e.Status >= status.Notified_HOUR {
		return
	}

	b, err := bot.GetBot()
	if err != nil {
		logger.WithError(err).Error("getting bot")
		return
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		logger.WithError(err).Error("getting bot")
		return
	}

	timer := time.NewTimer(time.Until(e.Start.Add(-1 * time.Hour)))
	<-timer.C

	if _, ok := scheduled[e.Id]; ok {
		if e.Status != status.Live {
			if message.Thread == nil {
				thread, err := b.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
					Name:                "Event Notification",
					Type:                discordgo.ChannelTypeGuildPrivateThread,
					AutoArchiveDuration: 60,
				})
				if err != nil {
					logger.WithError(err).Error("starting event thread")
					return
				}

				message.Thread = thread
			}

			e.Status = status.Notified_HOUR
			if err := e.Save(); err != nil {
				logger.WithError(err).Error("saving notification hour event")
				return
			}

			// alert the event is live
			participents := getAllParticipents(e)
			if participents != "" {
				if _, err := b.ChannelMessageSend(message.Thread.ID, participents+"\n\nReminder that this event is happening in an hour!"); err != nil {
					logger.WithError(err).Error("sending event starting message")
					return
				}
			}
		}
	}
}

func (e *Event) NotifyOfEvent() error {
	logger := log.WithField("method", "NotifyOfEvent")

	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Time",
			Value: fmt.Sprintf("<t:%d> - <t:%d:t>\n:timer: <t:%d:R>", e.Start.Unix(), e.End.Unix(), e.Start.Unix()),
		},
	}

	emojis := emoji.CodeMap()

	for _, position := range e.Positions {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s (0/%d)", emojis[":"+strings.ToLower(position.Emoji)+":"], position.Name, position.Max),
			Value:  "-",
			Inline: true,
		})
	}

	// add a section that says the rank limit to each positions
	limits := ""
	for _, position := range e.Positions {
		limits += position.Name + " - " + position.MinRank.String() + "\n"
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "Rank Limits",
		Value: limits,
	})

	if e.Cover == "/logo.png" || e.Cover == "" {
		e.Cover = "https://admin.solarmada.space/logo.png"
	}

	message, err := b.SendComplexMessage(config.GetString("DISCORD.CHANNELS.EVENTS"), &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeArticle,
				Title:       e.Name,
				Description: e.Description,
				Fields:      fields,
				Image: &discordgo.MessageEmbedImage{
					URL: e.Cover,
				},
			},
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

	for _, p := range e.Positions {
		if err := b.MessageReactionAdd(message.ChannelID, message.ID, emojis[":"+strings.ToLower(p.Emoji)+":"]); err != nil {
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

func EventWatcher() {
	logger := log.WithField("method", "EventWatcher")

	ticker := time.NewTicker(10 * time.Second)
	for {
		// get the events
		e := []*Event{}
		cur, err := stores.Storage.GetEvents(bson.M{"$and": []bson.M{
			{"start": bson.M{"$gte": time.Now()}},
			{"status": bson.M{"$lt": status.Live}},
		}})
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				logger.Debug("no upcomming events found")
				continue
			}

			logger.WithError(err).Error("getting the next event")
			continue
		}
		if err := cur.All(context.Background(), &e); err != nil {
			logger.WithError(err).Error("getting the next event")
			continue
		}

		for _, ev := range e {
			if _, ok := scheduled[ev.Id]; !ok {
				logger = logger.WithField("Event", ev.Id)
				logger.Debug("got event")

				if ev.MessageId == "" {
					logger.Debug("next event has no message associated, skipping this pass")
					continue
				}

				go ev.Schedule()
				scheduled[ev.Id] = ev
			}
		}

		<-ticker.C
	}
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
