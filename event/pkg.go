package event

import (
	"os/user"
	"time"

	"github.com/rs/xid"
	apierrors "github.com/sol-armada/admin/errors"
)

type Event struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	Start     time.Time    `json:"start_date"`
	Duration  int32        `json:"duration"`
	Attended  []*user.User `json:"attended"`
	Cancelled bool         `json:"cancelled"`
}

func New(body map[string]interface{}) (*Event, error) {

	name, ok := body["name"].(string)
	if !ok {
		return nil, apierrors.ErrMissingName
	}

	startRaw, ok := body["start"].(string)
	if !ok {
		return nil, apierrors.ErrMissingStart
	}

	start, err := time.Parse(time.RFC3339, startRaw)
	if err != nil {
		return nil, apierrors.ErrStartWrongFormat
	}

	durationRaw, ok := body["duration"].(float64)
	if !ok {
		return nil, apierrors.ErrMissingDuration
	}
	duration := int32(durationRaw)

	event := &Event{
		Id:        xid.New().String(),
		Name:      name,
		Start:     start,
		Duration:  duration,
		Attended:  []*user.User{},
		Cancelled: false,
	}

	return event, nil
}
