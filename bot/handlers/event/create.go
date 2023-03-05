package event

import (
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

func createEvent(s *discordgo.Session, i *discordgo.Interaction) {
	logger := log.WithField("func", "createEvent")
	now := time.Now()
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Type:        discordgo.EmbedTypeRich,
					Title:       "Test",
					Description: "Just a test event",
					Color:       1,
					Timestamp:   now.Format("2006-01-02 15:04:05.000"),
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Time",
							Value: fmt.Sprintf("<t:%d:R>", now.Unix()),
						},
					},
				},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("sending interaction embedded response")
	}
}
