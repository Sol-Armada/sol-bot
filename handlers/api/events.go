package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	apierrors "github.com/sol-armada/admin/errors"
	e "github.com/sol-armada/admin/events"
)

type CreateEventRequest struct {
	Name      string    `json:"name"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Repeat    int       `json:"repeat"`
	AutoStart bool      `json:"auto_start"`
	Positions map[string]struct {
		Name    string `json:"name"`
		Max     int32  `json:"max"`
		MinRank int32  `json:"min_rank"`
	} `json:"positions"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
}

type UpdateEventRequest struct {
	Id          string                            `json:"_id"`
	Name        string                            `json:"name"`
	Start       time.Time                         `json:"start"`
	End         time.Time                         `json:"end"`
	Repeat      int                               `json:"repeat"`
	AutoStart   bool                              `json:"auto_start"`
	Positions   map[string]map[string]interface{} `json:"positions"`
	Description string                            `json:"description"`
	Cover       string                            `json:"cover"`
}

type CreateEventResponse struct {
	Event *e.Event `json:"event"`
}

type getEventsResponse struct {
	Events []*e.Event `json:"events"`
}

func GetEvents(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetEvents",
	})
	logger.Debug("getting events")

	events, err := e.GetAll()
	if err != nil {
		logger.WithError(err).Error("getting events")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, getEventsResponse{
		Events: events,
	})
}

func (r *CreateEventRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}

func (r *UpdateEventRequest) bind(c echo.Context) error {
	if err := c.Bind(r); err != nil {
		return err
	}
	return nil
}

func (r *CreateEventRequest) toMap() (map[string]interface{}, error) {
	jsonRequest, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	mapRequest := map[string]interface{}{}
	if err := json.Unmarshal(jsonRequest, &mapRequest); err != nil {
		return nil, err
	}

	start, err := time.Parse(time.RFC3339, mapRequest["start"].(string))
	if err != nil {
		return nil, err
	}

	mapRequest["start"] = start

	end, err := time.Parse(time.RFC3339, mapRequest["end"].(string))
	if err != nil {
		return nil, err
	}

	mapRequest["end"] = end

	return mapRequest, nil
}

func (r *UpdateEventRequest) toMap() (map[string]interface{}, error) {
	jsonRequest, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	mapRequest := map[string]interface{}{}
	if err := json.Unmarshal(jsonRequest, &mapRequest); err != nil {
		return nil, err
	}

	start, err := time.Parse(time.RFC3339, mapRequest["start"].(string))
	if err != nil {
		return nil, err
	}

	mapRequest["start"] = start

	end, err := time.Parse(time.RFC3339, mapRequest["end"].(string))
	if err != nil {
		return nil, err
	}

	mapRequest["end"] = end

	return mapRequest, nil
}

func CreateEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "CreateEvent",
	})
	logger.Debug("creating event")

	req := &CreateEventRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, "internal server error")
	}

	reqMap, err := req.toMap()
	if err != nil {
		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	event, err := e.New(reqMap)
	if err != nil {
		if errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrMissingDuration) || errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrStartWrongFormat) || errors.Is(err, apierrors.ErrMissingId) {
			return c.JSON(http.StatusBadRequest, "internal server error")
		}

		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}
	if err := event.Save(); err != nil {
		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, CreateEventResponse{Event: event})
}

func UpdateEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "UpdateEvent",
	})
	logger.Debug("update event")

	req := &UpdateEventRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, "internal server error")
	}

	reqMap, err := req.toMap()
	if err != nil {
		logger.WithError(err).Error("update request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	event, err := e.Get(req.Id)
	if err != nil {
		logger.WithError(err).Error("getting event for update")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := event.Update(reqMap); err != nil {
		logger.WithError(err).Error("updating event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, CreateEventResponse{Event: event})
}

func DeleteEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "DeleteEvent",
	})

	params := c.ParamValues()
	logger.WithField("id", params[0]).Debug("deleting event")

	event, err := e.Get(params[0])
	if err != nil {
		if errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrMissingDuration) || errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrStartWrongFormat) || errors.Is(err, apierrors.ErrMissingId) {
			return c.JSON(http.StatusBadRequest, "internal server error")
		}

		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := event.Delete(); err != nil {
		logger.WithError(err).Error("deleting event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusOK)
}
