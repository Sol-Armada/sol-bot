package bot

import (
	"errors"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/mongo"
)

func (b *Bot) EventWatcher() {
	logger := log.WithField("method", "EventWatcher")

	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C

		// the next event got deleted
		if nextEvent != nil && !nextEvent.Exists() {
			logger.Debug(fmt.Sprintf("%s got deleted", nextEvent.Id))
			nextEvent = nil
			continue
		}

		// the next event is already set
		if nextEvent != nil {
			logger.WithField("NextEvent", nextEvent.Id).Debug("event already scheduled")
			continue
		}

		// get the next event
		if err := stores.Storage.GetNextEvent().Decode(&nextEvent); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				logger.Debug("no upcomming events found")
				continue
			}

			logger.WithError(err).Error("getting the next event")
			break
		}

		logger = logger.WithField("NextEvent", nextEvent.Id)

		if nextEvent.MessageId == "" {
			logger.Debug("next event has no message associated, skipping this pass")
			nextEvent = nil
			continue
		}

		// schedule the event
		logger.Debug("scheduling event")
		go b.ScheduleNextEvent()
	}
}
