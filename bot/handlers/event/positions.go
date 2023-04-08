package event

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/events"
)

func PositionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	eventId := strings.Split(i.MessageComponentData().CustomID, ":")[2]
	positionId := strings.Split(i.MessageComponentData().CustomID, ":")[3]
	message := i.Interaction.Message
	embedFeilds := message.Embeds[0].Fields

	logger := log.WithFields(log.Fields{
		"handler":  "EventPosition",
		"event":    eventId,
		"position": positionId,
		"member":   i.Member.User.Username,
	})

	event, err := events.Get(eventId)
	if err != nil {
		panic(err)
	}

	// remove them from all lists
	for _, ef := range embedFeilds[1:] {
		if ef.Value == "-" {
			continue
		}

		membersInPos := strings.Split(ef.Value, "\n")
		newMembersList := ""
		for _, m := range membersInPos {
			if m != i.Member.Nick && m != i.Member.User.Username {
				if i.Member.Nick == "" {
					newMembersList += i.Member.Nick
				} else {
					newMembersList += i.Member.User.ID
				}
				newMembersList += "\n"
			}
		}

		if newMembersList == "" {
			ef.Value = "-"
			continue
		}

		ef.Value = newMembersList
	}

	// check to see if they are in any other saved position
	// remove them if so
	for k, p := range event.Positions {
		newMembersList := []string{}
		for _, mip := range p.Members {
			if mip != i.Member.User.ID {
				newMembersList = append(newMembersList, mip)
				continue
			}
		}
		event.Positions[k].Members = newMembersList
	}

	// add them to the new position
	event.Positions[positionId].Members = append(event.Positions[positionId].Members, i.Member.User.ID)

	if err := event.Save(); err != nil {
		logger.WithError(err).Error("adding member to position")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}

	// update the message
	for _, ef := range embedFeilds {
		if ef.Name == event.Positions[positionId].Name {
			v := ""
			if ef.Value != "-" {
				v += ef.Value + "\n"
			}
			if i.Member.Nick != "" {
				v += i.Member.Nick
			} else {
				v += i.Member.User.Username
			}

			ef.Value = v
			break
		}
	}

	if _, err := s.ChannelMessageEditEmbeds(message.ChannelID, message.ID, message.Embeds); err != nil {
		logger.WithError(err).Error("updating embeds")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("You have been added the '%s' potision!", event.Positions[positionId].Name),
		},
	}); err != nil {
		logger.WithError(err).Error("responding")
		handlers.ErrorResponse(s, i.Interaction, "Ran into an issue in the backend. An officer should be here to help soon.")
		return
	}
}
