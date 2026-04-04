package blueprinthandler

import (
	"context"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

func removeHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData().Options[0]

	blueprintId := data.Options[0].StringValue()

	member, err := members.Get(i.Member.User.ID)
	if err != nil {
		return err
	}

	if !slices.Contains(member.BlueprintIds, blueprintId) {
		return nil
	}

	member.BlueprintIds = slices.DeleteFunc(member.BlueprintIds, func(blueprint string) bool {
		return blueprint == blueprintId
	})
	if err := member.Save(); err != nil {
		return err
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new("Blueprint removed from your profile!"),
	})
	return err
}

func removeAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("blueprint remove autocomplete handler")

	data := i.ApplicationCommandData()

	member, err := members.Get(i.Member.User.ID)
	if err != nil {
		return err
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, option := range data.Options {
		if option.Name == "remove" && option.Options[0].Focused {
			typed := option.Options[0].StringValue()
			if typed != "" {
				matches := fuzzy.FindFold(typed, member.BlueprintIds)

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
				for _, blueprintId := range member.BlueprintIds[:min(len(member.BlueprintIds), 24)] {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  blueprintId,
						Value: blueprintId,
					})
				}

				if len(member.BlueprintIds) > 24 {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  "Start typing to search (Showing first 24 blueprints)",
						Value: "NONE",
					})
				}
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
