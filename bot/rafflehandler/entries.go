package rafflehandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func entries(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle entries button")

	raffleIdStr := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	raffleId, err := strconv.Atoi(raffleIdStr)
	if err != nil {
		return err
	}

	raffle, err := raffles.Get(int32(raffleId))
	if err != nil {
		return err
	}

	tickets := raffle.Tickets[i.Member.User.ID]
	_ = tickets
	// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Flags:   discordgo.MessageFlagsEphemeral,
	// 		Content: fmt.Sprintf("You have %d tickets in the raffle", tickets),
	// 	},
	// }); err != nil {
	// 	return err
	// }

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new(fmt.Sprintf("You have %d tickets in the raffle", tickets)),
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return err
}
