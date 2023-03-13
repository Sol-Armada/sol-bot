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

type Position struct {
	Name string `json:"name" bson:"name"`
	Max  int32  `json:"max" bson:"max"`
}

type Event struct {
	Id          string       `json:"_id" bson:"_id"`
	Name        string       `json:"name" bson:"name"`
	Start       time.Time    `json:"start" bson:"start"`
	End         time.Time    `json:"end" bson:"end"`
	Repeat      Repeat       `json:"repeat" bson:"repeat"`
	AutoStart   bool         `json:"auto_start" bson:"auto_start"`
	Attendees   []*user.User `json:"attendees" bson:"attendees"`
	Status      Status       `json:"status" bson:"status"`
	Description string       `json:"description" bson:"description"`
	Cover       string       `json:"cover" bson:"cover"`
	Positions   []*Position  `json:"positions" bson:"positions"`
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

	positions := []*Position{}
	for k, v := range positionsRaw {
		positions = append(positions, &Position{
			Name: k,
			Max:  int32(v.(float64)),
		})
	}

	event := &Event{
		Id:          xid.New().String(),
		Name:        name,
		Start:       start,
		End:         end,
		Repeat:      Repeat(repeat),
		Attendees:   []*user.User{},
		Status:      Created,
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
	for k, v := range positionsRaw {
		positions = append(positions, &Position{
			Name: k,
			Max:  int32(v.(float64)),
		})
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

	return mapEvent
}
