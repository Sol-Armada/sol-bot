package blueprinthandler

import (
	"context"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/sol-bot/blueprints"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

func addHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData().Options[0]

	blueprintId := data.Options[0].StringValue()

	member, err := members.Get(i.Member.User.ID)
	if err != nil {
		return err
	}

	if slices.Contains(member.BlueprintIds, blueprintId) {
		return nil
	}

	member.BlueprintIds = append(member.BlueprintIds, blueprintId)
	if err := member.Save(); err != nil {
		return err
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new("Blueprint added to your profile!"),
	})
	return err
}

func addAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("blueprint add autocomplete handler")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	blueprintNames, err := blueprints.List()
	if err != nil {
		return err
	}

	for _, option := range data.Options {
		if slices.Contains([]string{"add", "list"}, option.Name) && option.Options[0].Focused {
			typed := option.Options[0].StringValue()
			if typed != "" {
				matches := fuzzy.FindFold(typed, blueprintNames)

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

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}
