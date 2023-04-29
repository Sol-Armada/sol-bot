package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/bot"
	apierrors "github.com/sol-armada/admin/errors"
	e "github.com/sol-armada/admin/events"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Position struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Max     int32  `json:"max"`
	MinRank int32  `json:"min_rank"`
	Emoji   string `json:"emoji"`
}

type CreateEventRequest struct {
	Name        string              `json:"name"`
	Start       time.Time           `json:"start"`
	End         time.Time           `json:"end"`
	Repeat      int                 `json:"repeat"`
	AutoStart   bool                `json:"auto_start"`
	Positions   map[string]Position `json:"positions"`
	Description string              `json:"description"`
	Cover       string              `json:"cover"`
}

type UpdateEventRequest struct {
	Id          string              `json:"_id"`
	Name        string              `json:"name"`
	Start       time.Time           `json:"start"`
	End         time.Time           `json:"end"`
	Repeat      int                 `json:"repeat"`
	AutoStart   bool                `json:"auto_start"`
	Positions   map[string]Position `json:"positions"`
	Description string              `json:"description"`
	Cover       string              `json:"cover"`
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

func CreateEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "CreateEvent",
	})
	logger.Debug("creating event")

	req := &CreateEventRequest{}
	if err := req.bind(c); err != nil {
		return c.JSON(http.StatusBadRequest, "internal server error")
	}

	if req.Cover == "" {
		req.Cover = "/logo.png"
	}

	reqMap, err := req.toMap()
	if err != nil {
		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	pTimeStart := primitive.NewDateTimeFromTime(reqMap["end"].(time.Time))
	pTimeEnd := primitive.NewDateTimeFromTime(reqMap["start"].(time.Time))

	cur, err := stores.Storage.GetEvents(
		bson.D{
			{
				Key: "$and",
				Value: bson.A{
					bson.D{{Key: "start", Value: bson.D{{Key: "$lt", Value: pTimeStart}}}},
					bson.D{{Key: "end", Value: bson.D{{Key: "$gt", Value: pTimeEnd}}}},
					bson.D{{Key: "status", Value: bson.D{{Key: "$gte", Value: 3}}}},
				},
			},
		},
	)
	if err != nil {
		logger.WithError(err).Error("getting mongo cursor")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	possibleOverlapEvents := []interface{}{}
	if err := cur.All(c.Request().Context(), &possibleOverlapEvents); err != nil {
		logger.WithError(err).Error("mongo result to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if len(possibleOverlapEvents) > 0 {
		return c.JSON(http.StatusConflict, "event overlaps existing event")
	}

	e, err := e.New(reqMap)
	if err != nil {
		if errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrMissingDuration) || errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrStartWrongFormat) || errors.Is(err, apierrors.ErrMissingId) {
			return c.JSON(http.StatusBadRequest, "internal server error")
		}

		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := e.Save(); err != nil {
		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := e.NotifyOfEvent(); err != nil {
		logger.WithError(err).Error("notifying of event")
	}

	return c.JSON(http.StatusOK, CreateEventResponse{Event: e})
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

	bot, err := bot.GetBot()
	if err != nil {
		logger.WithError(err).Error("getting bot for new event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := bot.DeleteEventMessage(event.MessageId); err != nil {
		logger.WithError(err).Error("deleting event message")
	}

	return c.NoContent(http.StatusOK)
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
