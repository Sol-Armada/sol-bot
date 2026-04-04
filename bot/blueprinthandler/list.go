package blueprinthandler

import (
	"context"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
)

func listHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData().Options[0]

	blueprintId := data.Options[0].StringValue()

	if blueprintId == "" {
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: new("Please provide a blueprint ID to search for."),
		})
		return err
	}

	members, err := members.ListByBlueprint(blueprintId)
	if err != nil {
		return err
	}

	var memberNames []string
	for _, member := range members {
		if slices.Contains(member.BlueprintIds, blueprintId) {
			memberNames = append(memberNames, "<@"+member.Id+">")
		}
	}

	var response strings.Builder
	response.WriteString("Members with blueprint " + blueprintId + ":\n")
	if len(memberNames) == 0 {
		response.WriteString("No members found with this blueprint.")
	} else {
		for _, name := range memberNames {
			response.WriteString("- " + name + "\n")
		}
	}

	// return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Content: response.String(),
	// 		Flags:   discordgo.MessageFlagsEphemeral,
	// 	},
	// })

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new(response.String()),
	})
	return err
}
