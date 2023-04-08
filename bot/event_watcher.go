package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/sol-armada/admin/events"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (b *Bot) EventWatcher() {
	logger := log.WithFields(log.Fields{
		"func": "EventWatcher",
	})
	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C

		if NextEvent != nil && !NextEvent.Exists() {
			logger.Debug(fmt.Sprintf("%s got deleted", NextEvent.Id))
			NextEvent = nil
			continue
		}

		// get all unannounced events
		unannouncedEvents := []events.Event{}
		cur, err := stores.Storage.GetEvents(bson.M{"status": 0})
		if err != nil {
			logger.WithError(err).Error("getting unannounced events")
			continue
		}

		if err := cur.All(context.Background(), &unannouncedEvents); err != nil {
			logger.WithError(err).Error("unmarshaling unannounced events")
			continue
		}

		if len(unannouncedEvents) > 0 {
			for _, event := range unannouncedEvents {
				event.Status = events.Announced
				event.Save()
			}
		}

		// get the next event
		nextStoredEvent := &events.Event{}
		if err := stores.Storage.GetNextEvent().Decode(&nextStoredEvent); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				logger.Debug("no upcomming schedules found")
				continue
			}

			logger.WithError(err).Error("getting the next event")
			break
		}

		if NextEvent == nil {
			logger.WithField("NextEvent", nextStoredEvent).Debug("scheduling event")

			// schedule the event
			go b.ScheduleEvent(nextStoredEvent)
		} else {
			if nextStoredEvent.Id != NextEvent.Id {
				logger.WithFields(log.Fields{
					"currentEvent": NextEvent,
					"soonerEvent":  nextStoredEvent,
				}).Debug("sooner event found")
				NextEvent.Timer.Stop()

				// schedule the event
				go b.ScheduleEvent(nextStoredEvent)
				continue
			}

			logger.Debug(fmt.Sprintf("an event is already scheduled: %s", NextEvent.Id))
		}
	}
}
