package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apex/log"
	apierrors "github.com/sol-armada/admin/errors"
	"github.com/sol-armada/admin/event"
	"github.com/sol-armada/admin/stores"
)

type CreateEventRequest struct {
	Name     string    `json:"name"`
	Start    time.Time `json:"start"`
	Duration int       `json:"duration"`
}

type CreateEventResponse struct {
	Event *event.Event `json:"event"`
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetEvents",
	})
	logger.Debug("getting events")

	// make sure we are only getting get
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storedEvents := []event.Event{}
	cur, err := stores.Storage.GetEvents()
	if err != nil {
		logger.WithError(err).Error("getting events")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := cur.All(r.Context(), &storedEvents); err != nil {
		logger.WithError(err).Error("getting events from collection")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jsonEvents, err := json.Marshal(storedEvents)
	if err != nil {
		logger.WithError(err).Error("marshaling events")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, string(jsonEvents)); err != nil {
		logger.WithError(err).Error("converting events to json string")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		if _, err := w.Write([]byte("Did not Work")); err != nil {
			log.WithError(err).Error("writting error response")
		}
		return
	}
	defer r.Body.Close()

	event, err := event.New(body)
	if err != nil {
		if errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrMissingDuration) || errors.Is(err, apierrors.ErrMissingStart) || errors.Is(err, apierrors.ErrStartWrongFormat) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error()))); err != nil {
				log.WithError(err).Error("write error response")
			}
			return
		}

		w.WriteHeader(500)
		if _, err := w.Write([]byte("internal server error")); err != nil {
			log.WithError(err).Error("write error response")
		}
		return
	}

	response := CreateEventResponse{
		Event: event,
	}

	responseJson, _ := json.Marshal(response)

	w.WriteHeader(200)
	if _, err := w.Write(responseJson); err != nil {
		log.WithError(err).Error("writting response")
	}
}
