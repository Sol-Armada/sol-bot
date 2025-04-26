package giveawayhandler

import (
	"context"
	"errors"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func startAutocomplete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("giveaway start autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	items := utils.GetItemNames()
	for _, option := range data.Options {
		if option.Name == "event" && option.Focused {
			attendanceRecords, err := attendance.ListActive(5)
			if err != nil {
				return errors.Join(err, errors.New("getting active attendance records"))
			}

			for _, record := range attendanceRecords {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  record.Name,
					Value: record.Id,
				})
			}
		}
		if strings.Contains(option.Name, "item") && option.Focused {
			typed := option.StringValue()
			if typed != "" {
				matches := fuzzy.FindFold(typed, items)

				for _, name := range matches {
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
