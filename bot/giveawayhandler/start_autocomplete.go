package giveawayhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func startAutocomplete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway start autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	items := utils.GetItemNames()
	for _, option := range data.Options {
		if option.Name == "event" && option.Focused {
			attendanceRecords, err := attendance.List(bson.D{}, 10, 0)
			if err != nil {
				return errors.Join(err, errors.New("getting active attendance records"))
			}

			for _, record := range attendanceRecords {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  fmt.Sprintf("%s (%s)", record.Name, record.DateCreated.Format("2006-01-02")),
					Value: record.Id,
				})
			}
		}
		if strings.Contains(option.Name, "item") && option.Focused {
			typed := option.StringValue()
			if typed != "" {
				typedS := strings.Split(typed, ":")
				typed = typedS[0]

				matches := fuzzy.Find(typed, items)

				for _, name := range matches {
					if len(typedS) > 1 {
						name = name + ":" + typedS[1]
					}

					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  name,
						Value: name,
					})

					if len(choices) >= 25 {
						break
					}
				}
			} else {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  "Start typing to search",
					Value: "NONE",
				})
			}
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Join(err, errors.New("responding to raffle start autocomplete"))
	}

	return nil
}
