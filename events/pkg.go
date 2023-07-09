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
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events/status"
	"github.com/sol-armada/admin/health"
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

type CommsTier int

const (
	Relaxed CommsTier = iota // no comm restrictions
	One                      // max PTT restrictions
	Two                      // mid PTT restrictions
	Three                    // low PTT restrictions
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
	Id          string        `json:"id" bson:"_id"`
	Name        string        `json:"name" bson:"name"`
	StartTime   time.Time     `json:"start_time" bson:"start_time"`
	EndTime     time.Time     `json:"end_time" bson:"end_time"`
	Repeat      Repeat        `json:"repeat" bson:"repeat"`
	AutoStart   bool          `json:"auto_start" bson:"auto_start"`
	Attendees   []*user.User  `json:"attendees" bson:"attendees"`
	Status      status.Status `json:"status" bson:"status"`
	Description string        `json:"description" bson:"description"`
	Cover       string        `json:"cover" bson:"cover"`
	Positions   []*Position   `json:"positions" bson:"positions"`
	MessageId   string        `json:"message_id" bson:"message_id"`
	PTU         bool          `json:"ptu" bson:"ptu"`
	CommsTier   CommsTier     `json:"comms_tier" bson:"comms_tier"`

	cancel chan bool
	mu     sync.Mutex
}

var scheduledEvents = map[string]*Event{}

func Get(id string) (*Event, error) {
	event := &Event{}
	eventRes := stores.Storage.GetEvent(id)
	if err := eventRes.Decode(&event); err != nil {
		return nil, err
	}

	return event, nil
}

func GetAll() ([]*Event, error) {
	cur, err := stores.Storage.GetEvents(bson.D{
		{
			Key: "$and",
			Value: bson.A{
				bson.D{{Key: "end_time", Value: bson.D{{Key: "$gte", Value: time.Now().AddDate(0, 0, -17).UnixMilli()}}}},
				bson.D{{Key: "status", Value: bson.D{{Key: "$lt", Value: status.Deleted}}}},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	eventsMap := []map[string]interface{}{}
	if err := cur.All(context.Background(), &eventsMap); err != nil {
		return nil, err
	}

	e := []*Event{}
	for _, eventMap := range eventsMap {
		event := &Event{}
		if err := event.parse(eventMap); err != nil {
			return nil, err
		}
		e = append(e, event)
	}

	return e, nil
}

func GetByMessageId(messageId string) (*Event, error) {
	cur, err := stores.Storage.GetEvents(bson.D{{Key: "message_id", Value: messageId}})
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
	cur, err := stores.Storage.GetEvents(filter)
	if err != nil {
		return nil, err
	}

	eventsMap := []map[string]interface{}{}
	if err := cur.All(context.Background(), &eventsMap); err != nil {
		return nil, err
	}

	e := []*Event{}
	for _, eventMap := range eventsMap {
		event := &Event{}
		if err := event.parse(eventMap); err != nil {
			return nil, err
		}
		e = append(e, event)
	}

	return e, nil
}

func GetAllParticipents(e *Event) string {
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
		if !health.IsHealthy() {
			logger.Error("not healthy")
			<-ticker.C
			continue
		}
		// get the events
		events, err := GetAllWithFilter(
			bson.D{{Key: "status", Value: bson.D{{Key: "$lt", Value: status.Cancelled}}}},
		)
		if err != nil {
			logger.WithError(err).Error("getting all events")
			<-ticker.C
			continue
		}

		// look over the events
		for _, e := range events {
			logger = logger.WithField("event_id", e.Id)

			if err := e.UpdateMessage(); err != nil {
				logger.WithError(err).Error("updated event message")
				continue
			}

			if scheduledEvents[e.Id] == nil {
				go e.schedule()
			}
		}

		<-ticker.C
	}
}

func ResetSchedule(e *Event) {
	event, ok := scheduledEvents[e.Id]
	if ok && event.cancel != nil {
		event.cancel <- true
		delete(scheduledEvents, e.Id)
	}
	go e.schedule()
}

func CancelSchedule(e *Event) {
	event, ok := scheduledEvents[e.Id]
	if ok && event.cancel != nil {
		event.cancel <- true
		delete(scheduledEvents, e.Id)
	}
}

func (e *Event) parse(eventMap map[string]interface{}) error {
	eventMap["id"] = eventMap["_id"]
	if startTime, ok := eventMap["start_time"].(float64); ok {
		eventMap["start_time"] = time.UnixMilli(int64(startTime))
	}
	if startTime, ok := eventMap["start_time"].(int64); ok {
		eventMap["start_time"] = time.UnixMilli(startTime)
	}
	if endTime, ok := eventMap["end_time"].(float64); ok {
		eventMap["end_time"] = time.UnixMilli(int64(endTime))
	}
	if endTime, ok := eventMap["end_time"].(int64); ok {
		eventMap["end_time"] = time.UnixMilli(endTime)
	}

	jsonEventMap, err := json.Marshal(eventMap)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonEventMap, &e); err != nil {
		return err
	}

	return nil
}

func (e *Event) schedule() {
	logger := log.WithFields(log.Fields{
		"event":  e.Id,
		"method": "schedule",
	})
	logger.Debug("scheduling")
	if e.cancel == nil {
		e.cancel = make(chan bool)
		defer close(e.cancel)
	}

	wg := sync.WaitGroup{}

	if e.Status < status.Notified_DAY {
		wg.Add(1)
		go func() {
			logger := logger.WithField("notification", "day")

			dayBefore := e.StartTime.UTC().AddDate(0, 0, -1)
			timer := time.NewTimer(time.Until(dayBefore))
			logger.WithField("until", dayBefore.Format(time.RFC3339)).Debug("waiting for day before event")
			select {
			case <-timer.C:
				if err := e.remindParticipents("Reminder that this event is happening tomorrow!"); err != nil {
					logger.WithError(err).Error("reminding participents")
					return
				}

				if err := e.SetStatus(status.Notified_DAY); err != nil {
					logger.WithError(err).Error("setting status to notified day")
				}
			case <-e.cancel:
				logger.WithField("event_id", e.Id).Debug("schedule got canceled")
			}
		}()
	}

	if e.Status < status.Notified_HOUR {
		wg.Add(1)
		go func() {
			logger := logger.WithField("notification", "hour")

			hourBefore := e.StartTime.UTC().Add(-1 * time.Hour)
			timer := time.NewTimer(time.Until(hourBefore))
			logger.WithField("until", hourBefore.Format(time.RFC3339)).Debug("waiting for hour before event")
			select {
			case <-timer.C:
				if err := e.remindParticipents("Reminder that this event is happening in an hour!"); err != nil {
					logger.WithError(err).Error("reminding participents")
					return
				}

				if err := e.SetStatus(status.Notified_HOUR); err != nil {
					logger.WithError(err).Error("setting status to notified hour")
				}
			case <-e.cancel:
				logger.WithField("event_id", e.Id).Debug("schedule got canceled")
			}
		}()
	}

	if e.Status < status.Live {
		wg.Add(1)
		go func() {
			logger := logger.WithField("notification", "live")

			live := e.StartTime.UTC()
			timer := time.NewTimer(time.Until(live))
			logger.WithField("until", live.Format(time.RFC3339)).Debug("waiting for live event")
			select {
			case <-timer.C:
				if err := e.remindParticipents("This event is live!"); err != nil {
					logger.WithError(err).Error("reminding participents")
					return
				}

				if err := e.SetStatus(status.Live); err != nil {
					logger.WithError(err).Error("setting status to live")
					return
				}

				if err := e.UpdateMessage(); err != nil {
					if strings.Contains(err.Error(), "Unknown Message") {
						e.MessageId = ""
						if err := e.Save(); err != nil {
							logger.WithError(err).Error("saving event on unknown message")
							return
						}
						return
					}
					logger.WithError(err).Error("updating message")
					return
				}
			case <-e.cancel:
				logger.Debug("schedule got canceled")
			}
		}()
	}

	if e.Status < status.Finished {
		wg.Add(1)
		go func() {
			logger := logger.WithField("notification", "end")

			fin := e.EndTime.UTC()
			timer := time.NewTimer(time.Until(fin))
			logger.WithField("until", fin.Format(time.RFC3339)).Debug("waiting for end of event")
			select {
			case <-timer.C:
				if err := e.UpdateMessage(); err != nil {
					if strings.Contains(err.Error(), "Unknown Message") {
						e.MessageId = ""
						if err := e.Save(); err != nil {
							logger.WithError(err).Error("saving event on unknown message")
							return
						}
						return
					}
					logger.WithError(err).Error("updating message")
					return
				}

				e.Status = status.Finished
				if err := e.Save(); err != nil {
					logger.WithError(err).Error("saving event")
					return
				}

				break
			case <-e.cancel:
				logger.Debug("schedule got canceled")
			}
		}()
	}

	if e.Status < status.Deleted {
		wg.Add(1)
		go func() {
			logger := logger.WithField("notification", "delete")

			fin := e.EndTime.UTC().AddDate(0, 0, 1)
			timer := time.NewTimer(time.Until(fin))
			logger.WithField("until", fin.Format(time.RFC3339)).Debug("waiting for deletion of event")
			select {
			case <-timer.C:
				logger.Debug("deleting event")

				if err := e.Delete(); err != nil {
					logger.WithError(err).Error("deleting event")
				}
				delete(scheduledEvents, e.Id)
				break
			case <-e.cancel:
				logger.Debug("schedule got canceled")
			}
		}()
	}

	scheduledEvents[e.Id] = e

	wg.Wait()
	logger.Debug("Event over")
}

func (e *Event) SetStatus(s status.Status) error {
	e.Lock()
	e.Status = s
	e.Unlock()

	if err := e.Save(); err != nil {
		return err
	}

	return nil
}

func (e *Event) Lock() {
	e.mu.Lock()
}

func (e *Event) Unlock() {
	e.mu.Unlock()
}

func (e *Event) Save() error {
	e.Lock()
	defer e.Unlock()
	return stores.Storage.SaveEvent(e.ToMap())
}

func (e *Event) Delete() error {
	bot, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot for new event")
	}

	if err := bot.DeleteEventMessage(e.MessageId); err != nil {
		if !strings.Contains(err.Error(), "404") {
			return errors.Wrap(err, "deleting event message")
		}
	}

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

	mapEvent["start_time"] = e.StartTime.UnixMilli()
	mapEvent["end_time"] = e.EndTime.UnixMilli()

	return mapEvent
}

func (e *Event) Exists() bool {
	return stores.Storage.GetEvent(e.Id).Err() == nil
}

func (e *Event) RemoveReactions() error {
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	if err := e.SetStatus(status.Live); err != nil {
		return errors.Wrap(err, "settings event status")
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting original event message")
	}

	if err := b.MessageReactionsRemoveAll(message.ChannelID, message.ID); err != nil {
		return errors.Wrap(err, "removing all emojis")
	}

	return nil
}

func (e *Event) UpdateMessage() error {
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting original event message")
	}

	embeds, err := e.GetEmbeds()
	if err != nil {
		return errors.Wrap(err, "getting embeds")
	}

	buttons, err := e.GetButtons()
	if err != nil {
		return errors.Wrap(err, "getting buttons")
	}

	if e.Status >= status.Live {
		buttons = nil
	}

	if _, err := b.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      message.ID,
		Channel: message.ChannelID,
		Embeds: []*discordgo.MessageEmbed{
			embeds,
		},
		Components: buttons,
	}); err != nil {
		return errors.Wrap(err, "updating original event message")
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

	participents := GetAllParticipents(e)
	if participents != "" {
		if _, err := b.ChannelMessageSend(message.Thread.ID, participents+"\n\n"+msg); err != nil {
			return errors.Wrap(err, "sending reminder message")
		}
	}

	return nil
}

func (e *Event) getTimeField() string {
	timeField := fmt.Sprintf("<t:%d> - <t:%d:t>\n:timer: <t:%d:R>", e.StartTime.Unix(), e.EndTime.Unix(), e.StartTime.Unix())
	if e.Status == status.Live {
		timeField = fmt.Sprintf("<t:%d> - <t:%d:t>\nLive!", e.StartTime.Unix(), e.EndTime.Unix())
	}
	if e.Status > status.Live {
		timeField = fmt.Sprintf("<t:%d> - <t:%d:t>", e.StartTime.Unix(), e.EndTime.Unix())
	}

	return timeField
}

func (e *Event) AllPositionsFilled() bool {
	for _, position := range e.Positions[1:] {
		if strings.Contains(position.Name, "Extras") {
			continue
		}
		if len(position.Members) < int(position.Max) {
			return false
		}
	}

	return true
}

func (e *Event) GetEmbeds() (*discordgo.MessageEmbed, error) {
	e.Lock()
	defer e.Unlock()

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
		sort.Slice(position.Members, func(i, j int) bool {
			mI, err := user.Get(position.Members[i])
			if err != nil {
				return false
			}
			mJ, err := user.Get(position.Members[j])
			if err != nil {
				return false
			}

			return mI.Rank < mJ.Rank
		})

		emoji := emojis[strings.ToLower(position.Emoji)]
		b, _ := bot.GetBot()
		customEmojis, _ := b.GetEmojis()
		for _, customEmoji := range customEmojis {
			if strings.Contains(position.Emoji, customEmoji.Name) {
				emoji = customEmoji.MessageFormat()
			}
		}

		name := fmt.Sprintf("%s %s (%d/%d)", emoji, position.Name, len(position.Members), position.Max)
		names := ""

		if position.Max == 0 {
			name = fmt.Sprintf("%s %s (%d/-)", emojis[strings.ToLower(position.Emoji)], position.Name, len(position.Members))
			if position.FillLast {
				names = "fill others first"
			}
			if e.AllPositionsFilled() {
				names = "Open"
			}
			goto SKIP
		}

		for _, m := range position.Members {
			u, err := user.Get(m)
			if err != nil {
				return nil, err
			}

			// check if they have a nickname
			if u.Discord != nil && u.Discord.Nick != "" {
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
		Name:  "Minimum Rank Requirements per Position",
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
	b, err := bot.GetBot()
	if err != nil {
		return errors.Wrap(err, "getting bot")
	}

	embeds, err := e.GetEmbeds()
	if err != nil {
		return err
	}

	buttons, err := e.GetButtons()
	if err != nil {
		return err
	}

	message, err := b.SendComplexMessage(config.GetString("DISCORD.CHANNELS.EVENTS"), &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embeds},
		Components: buttons,
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

	e.MessageId = message.ID
	eventStatus := status.Announced

	if time.Now().UTC().After(e.StartTime.AddDate(0, 0, -1)) {
		eventStatus = status.Notified_DAY
	}

	if time.Now().UTC().After(e.StartTime.Add(-1 * time.Hour)) {
		eventStatus = status.Notified_HOUR
	}

	if err := e.SetStatus(eventStatus); err != nil {
		return err
	}

	return nil
}

func (e *Event) GetButtons() ([]discordgo.MessageComponent, error) {
	components := []discordgo.MessageComponent{}
	subComponents := &discordgo.ActionsRow{}
	e.Lock()
	defer e.Unlock()
	for i, pos := range e.Positions {
		if i%5 == 0 {
			subComponents = &discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{},
			}
			components = append(components, subComponents)
		}

		posEmoji := discordgo.ComponentEmoji{
			Name: strings.TrimSpace(emoji.Sprint(pos.Emoji)),
		}

		// is custom
		if strings.HasPrefix(posEmoji.Name, ":") {
			b, err := bot.GetBot()
			if err != nil {
				return nil, err
			}

			customEmojis, err := b.GetEmojis()
			if err != nil {
				return nil, err
			}

			for _, customEmoji := range customEmojis {
				if strings.Contains(posEmoji.Name, customEmoji.Name) {
					posEmoji.Name = customEmoji.Name
					posEmoji.ID = customEmoji.ID
					posEmoji.Animated = customEmoji.Animated
					break
				}
			}
		}

		subComponents.Components = append(subComponents.Components, discordgo.Button{
			Emoji:    posEmoji,
			CustomID: fmt.Sprintf("event:choice:%s:%s", e.Id, pos.Id),
		})
	}
	return components, nil
}
