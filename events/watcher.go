package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func EventWatcher() {
	logger := log.WithFields(log.Fields{
		"func": "EventWatcher",
	})
	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C

		// get all unannounced events
		unannouncedEvents := []Event{}
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
			b, err := bot.GetBot()
			if err != nil {
				logger.WithError(err).Error("getting the bot")
				continue
			}

			for _, event := range unannouncedEvents {
				fields := []*discordgo.MessageEmbedField{
					{
						Name:  "Time",
						Value: fmt.Sprintf("<t:%d> - <t:%d:t>\n<t:%d:R>", event.Start.Unix(), event.End.Unix(), event.Start.Unix()),
					},
				}

				buttons := []discordgo.MessageComponent{}

				for i, pos := range event.Positions {
					fields = append(fields, &discordgo.MessageEmbedField{
						Name:   fmt.Sprintf("%s (0/%d)", pos.Name, pos.Max),
						Value:  "",
						Inline: true,
					})

					logger.Debug(fmt.Sprintf("%d", i%3))
					// row := discordgo.ActionsRow{
					// 	Components: []discordgo.MessageComponent{},
					// }

					// buttons = append(buttons, row)
				}

				embeds := []*discordgo.MessageEmbed{
					{
						Title:       event.Name,
						Description: event.Description,
						Fields:      fields,
						Image: &discordgo.MessageEmbedImage{
							URL: event.Cover,
						},
					},
				}

				if _, err := b.SendComplexMessage(config.GetString("DISCORD.CHANNELS.ANNOUNCEMENTS"), &discordgo.MessageSend{
					Embeds:     embeds,
					Components: buttons,
				}); err != nil {
					logger.WithError(err).Error("sending the announcement")
				}

				event.Status = Announced
				event.Save()
			}
		}

		// get the next event
		nextStoredEvent := Event{}
		if err := stores.Storage.GetNextEvent().Decode(&nextStoredEvent); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				logger.Debug("no upcomming schedules found")
				continue
			}

			logger.WithError(err).Error("getting the next event")
			break
		}

		if nextEvent == nil {
			logger.WithField("nextEvent", nextStoredEvent).Debug("scheduling event")

			// schedule the event
			go nextStoredEvent.Schedule()
		} else {
			if nextStoredEvent.Id != nextEvent.Id {
				logger.WithFields(log.Fields{
					"currentEvent": nextEvent,
					"soonerEvent":  nextStoredEvent,
				}).Debug("sooner event found")
				nextEvent.Timer.Stop()

				// schedule the event
				go nextStoredEvent.Schedule()
				continue
			}

			logger.Debug(fmt.Sprintf("an event is already scheduled: %s", nextEvent.Id))
		}
	}
}
