package event

import (
	"encoding/json"
	"os/user"
	"time"

	"github.com/apex/log"
	"github.com/rs/xid"
	apierrors "github.com/sol-armada/admin/errors"
	"github.com/sol-armada/admin/stores"
)

type Repeat int

const (
	None Repeat = iota
	Daily
	Weekly
	Monthly
)

type Status int

const (
	Created Status = iota
	Announced
	Live
	Finished
	Cancelled
)

type Event struct {
	Id        string       `json:"_id" bson:"_id"`
	Name      string       `json:"name" bson:"name"`
	Start     time.Time    `json:"start_date" bson:"start_date"`
	Duration  float64      `json:"duration" bson:"duration"`
	Repeat    Repeat       `json:"repeat" bson:"repeat"`
	AutoStart bool         `json:"auto_start" bson:"auto_start"`
	Attendees []*user.User `json:"attendees" bson:"attendees"`
	Status    Status       `json:"status" bson:"status"`
}

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

	duration := end.Sub(start)

	repeatRaw, ok := body["repeat"].(float64)
	if !ok {
		repeatRaw = 0
	}
	repeat := int32(repeatRaw)

	autoStart, ok := body["auto_start"].(bool)
	if !ok {
		autoStart = false
	}

	event := &Event{
		Id:        xid.New().String(),
		Name:      name,
		Start:     start,
		Duration:  duration.Minutes(),
		Repeat:    Repeat(repeat),
		Attendees: []*user.User{},
		Status:    Created,
		AutoStart: autoStart,
	}
	if err := event.Save(); err != nil {
		return nil, err
	}

	return event, nil
}

func (e *Event) Save() error {
	return stores.Storage.SaveEvent(e.ToMap())
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

	return mapEvent
}
