package rafflehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
)

func end(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle end button")

	// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Flags: discordgo.MessageFlagsEphemeral,
	// 	},
	// }); err != nil {
	// 	return err
	// }

	if !utils.Allowed(i.Member, "RAFFLES") {
		_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "You do not have the permissions to do that.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	data := i.MessageComponentData()

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return err
	}

	winners, err := raffle.PickWinner()
	if err != nil {
		if err == raffles.ErrNoEntries {
			raffle.Ended = true
			if err := raffle.Save(); err != nil {
				return err
			}
		}

		return raffle.UpdateMessage(s)
	}

	var winnerNames strings.Builder
	for j, winner := range winners {
		winnerNames.WriteString("<@")
		winnerNames.WriteString(winner.Id)
		winnerNames.WriteString(">")
		if j < len(winners)-1 {
			if j == len(winners)-2 {
				winnerNames.WriteString(" and ")
			}
			winnerNames.WriteString(", ")
		}

		if !raffle.Test {
			if err := tokens.New(winner.Id, raffle.Tickets[winner.Id]*-1, tokens.ReasonWonRaffle, nil, &raffle.AttedanceId, nil).Save(); err != nil {
				return err
			}
		}
	}

	if _, err := s.ChannelMessageSend(i.Interaction.ChannelID, fmt.Sprintf("🎊 Congratulations to %s! They have won the raffle! 🎊", winnerNames.String())); err != nil {
		return err
	}

	return raffle.UpdateMessage(s)
}
