package events

import (
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/user"
)

// event command handlers
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){}

// event interaction handlers
var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice": ChoiceButtonHandler,
}

func Setup(b *bot.Bot) error {
	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			id := strings.Split(i.MessageComponentData().CustomID, ":")

			if id[0] == "event" {
				interactionHandlers[id[1]](s, i)
			}
		}
	})

	// watch the events
	go EventWatcher()

	return nil
}

func ChoiceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := log.WithField("func", "ChoiceButtonHandler")

	event, err := Get(strings.Split(i.MessageComponentData().CustomID, ":")[2])
	if err != nil {
		logger.WithError(err).Error("getting event")
		return
	}

	posId := strings.Split(i.MessageComponentData().CustomID, ":")[3]

	user, err := user.Get(i.Member.User.ID)
	if err != nil {
		logger.WithError(err).Error("getting user")
		return
	}

	event.Lock()
	for _, pos := range event.Positions {
		if pos.Id == posId {
			if pos.FillLast && !event.AllPositionsFilled() {
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "This position can only be filled when all others are filled. Please select a different one.",
					},
				}); err != nil {
					logger.WithError(err).Error("fill other positions response")
				}

				continue
			}

			if pos.MinRank < user.Rank {
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "You don't meet the minimum rank requirements for this position. Please select a different one.",
					},
				}); err != nil {
					logger.WithError(err).Error("min rank response")
				}

				return
			}
		}

		members := []string{}
		justRemove := false
		// first remove this user from all positions
		for _, memberId := range pos.Members {
			// remove them from the list and move on
			if memberId == i.Member.User.ID && pos.Id == posId {
				justRemove = true

				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   discordgo.MessageFlagsEphemeral,
						Content: "You have been removed from the " + pos.Name + " position.",
					},
				}); err != nil {
					logger.WithError(err).Error("sending response")
				}
				continue
			}

			if memberId == i.Member.User.ID {
				continue
			}
			members = append(members, memberId)
		}
		// add this user the selected position
		if pos.Id == posId && !justRemove {
			members = append(members, i.Member.User.ID)

			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: "You have been added to the " + pos.Name + " position!",
				},
			}); err != nil {
				logger.WithError(err).Error("sending response")
			}
		}
		pos.Members = members
	}
	event.Unlock()

	go func() {
		time.Sleep(15 * time.Second)
		s.InteractionResponseDelete(i.Interaction)
	}()

	if err := event.UpdateMessage(); err != nil {
		logger.WithError(err).Error("updating event message")
		return
	}

	if err := event.Save(); err != nil {
		logger.WithError(err).Error("saving event")
		return
	}
}
