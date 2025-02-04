package tokenshandler

import (
	"context"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
)

func reasonAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("reason autocomplete handler")

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Attended Event",
					Value: tokens.ReasonAttendance,
				},
				{
					Name:  "Attended Full Event",
					Value: tokens.ReasonAttendanceFull,
				},
				{
					Name:  "Event Succeded",
					Value: tokens.ReasonEventSuccessful,
				},
				{
					Name:  "Other",
					Value: tokens.ReasonOther,
				},
			},
		},
	})
}
