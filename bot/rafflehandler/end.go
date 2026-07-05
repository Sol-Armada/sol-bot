package rafflehandler

import (
	"context"
	"fmt"
	"strconv"
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

	data := i.MessageComponentData()

	raffleIdStr := strings.Split(data.CustomID, ":")[2]

	raffleId, err := strconv.Atoi(raffleIdStr)
	if err != nil {
		return err
	}

	raffle, err := raffles.Get(int32(raffleId))
	if err != nil {
		return err
	}

	winners, err := raffle.PickWinner(ctx)
	if err != nil {
		if err == raffles.ErrNoEntries {
			raffle.Ended = true
			if err := raffle.Save(ctx); err != nil {
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
			if err := tokens.New(winner.Id, raffle.Tickets[winner.Id]*-1, tokens.ReasonWonRaffle, nil, &raffle.AttendanceId, nil).Save(); err != nil {
				return err
			}
		}
	}

	if _, err := s.ChannelMessageSend(i.Interaction.ChannelID, fmt.Sprintf("🎊 Congratulations to %s! They have won the raffle! 🎊", winnerNames.String())); err != nil {
		return err
	}

	if err := s.InteractionResponseDelete(i.Interaction); err != nil {
		return err
	}

	return raffle.UpdateMessage(s)
}
