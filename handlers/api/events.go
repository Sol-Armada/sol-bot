package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	apierrors "github.com/sol-armada/sol-bot/errors"
	"github.com/sol-armada/sol-bot/events"
	e "github.com/sol-armada/sol-bot/events"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type Position struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Max      int32  `json:"max"`
	MinRank  int32  `json:"min_rank"`
	Emoji    string `json:"emoji"`
	Order    int32  `json:"order"`
	FillLast bool   `json:"fill_last"`
}

type CreateEventResponse struct {
	Event *e.Event `json:"event"`
}

func CreateEvent(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "CreateEvent",
	})
	logger.Debug("creating event")

	event := &events.Event{
		Id: xid.New().String(),
	}
	if err := c.Bind(&event); err != nil {
		logger.WithError(err).Error("binding to event")
		return c.JSON(http.StatusBadRequest, "bad request")
	}

	if event.StartTime.Before(time.Now().UTC()) {
		return c.JSON(http.StatusBadRequest, "{\"error\": \"start time must be in the future\", \"code\": 2}")
	}

	if event.Cover == "" {
		event.Cover = "/logo.png"
	}

	pTimeStart := primitive.NewDateTimeFromTime(event.StartTime)
	pTimeEnd := primitive.NewDateTimeFromTime(event.EndTime)

	if event.StartTime.After(event.EndTime) {
		return c.JSON(http.StatusBadRequest, "{\"error\": \"start time must be before end time\", \"code\": 2}")
	}

	store := stores.Events
	cur, err := store.List(bson.D{
		{
			Key: "$and",
			Value: bson.A{
				bson.D{{Key: "start", Value: bson.D{{Key: "$lt", Value: pTimeStart}}}},
				bson.D{{Key: "end", Value: bson.D{{Key: "$gt", Value: pTimeEnd}}}},
				bson.D{{Key: "status", Value: bson.D{{Key: "$gte", Value: 3}}}},
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("getting events cursor")
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

	// add ids to the positions
	for _, p := range event.Positions {
		p.Id = xid.New().String()
	}

	if err := event.NotifyOfEvent(); err != nil {
		logger.WithError(err).Error("notifying of event")
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

	event := &events.Event{}
	if err := c.Bind(&event); err != nil {
		logger.WithError(err).Error("binding to event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := event.Save(); err != nil {
		logger.WithError(err).Error("saving event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := event.UpdateMessage(); err != nil {
		logger.WithError(err).Error("updating message")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	events.ResetSchedule(event)

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

	events.CancelSchedule(event)

	return c.NoContent(http.StatusOK)
}

type CreateEventTemplateResponse struct {
	Template *e.Template `json:"template"`
}

func CreateEventTemplate(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "CreateEventTemplate",
	})
	logger.Debug("creating event template")

	template := &events.Template{}

	if err := c.Bind(&template); err != nil {
		logger.WithError(err).Error("binding to event template")
		return c.JSON(http.StatusBadRequest, "bad request")
	}

	exists, err := events.TemplateExists(template.Name)
	if err != nil {
		logger.WithError(err).Error("checking if template exists")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if exists {
		return c.JSON(http.StatusConflict, "template already exists")
	}

	if template.Cover == "" {
		template.Cover = "/logo.png"
	}

	if err := template.Save(); err != nil {
		logger.WithError(err).Error("request to map")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, CreateEventTemplateResponse{Template: template})
}

func GetEventTemplates(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetEventTemplates",
	})
	logger.Debug("getting event templates")

	templates, err := events.GetAllTemplates()
	if err != nil {
		logger.WithError(err).Error("getting event templates")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, templates)
}

func UpdateEventTemplate(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "UpdateEventTemplate",
	})
	logger.Debug("update event template")

	template := &events.Event{}

	if err := c.Bind(&template); err != nil {
		logger.WithError(err).Error("binding to event template")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := template.Save(); err != nil {
		logger.WithError(err).Error("saving event template")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusOK)
}

func DeleteEventTemplate(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "DeleteEventTemplate",
	})
	logger.Debug("deleting event template")

	template := &events.Event{}

	if err := c.Bind(&template); err != nil {
		logger.WithError(err).Error("binding to event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if err := template.Delete(); err != nil {
		logger.WithError(err).Error("deleting event")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusOK)
}
