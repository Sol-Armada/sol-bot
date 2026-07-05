package rafflehandler

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func backOut(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle back out button")

	// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Flags: discordgo.MessageFlagsEphemeral,
	// 	},
	// }); err != nil {
	// 	return errors.Join(err, errors.New("responding to back out button"))
	// }

	data := i.MessageComponentData()

	raffleIdStr := strings.Split(data.CustomID, ":")[2]

	raffleId, err := strconv.Atoi(raffleIdStr)
	if err != nil {
		return errors.Join(err, errors.New("parsing raffle ID"))
	}

	raffle, err := raffles.Get(int32(raffleId))
	if err != nil {
		return errors.Join(err, errors.New("getting raffle"))
	}

	if err := raffle.RemoveTicket(i.Member.User.ID).Save(ctx); err != nil {
		return errors.Join(err, errors.New("removing ticket"))
	}

	// if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
	// 	Content: "You have successfully backed out of the raffle.",
	// 	Flags:   discordgo.MessageFlagsEphemeral,
	// }); err != nil {
	// 	return errors.Join(err, errors.New("creating followup message"))
	// }

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new("You have successfully backed out of the raffle."),
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return errors.Join(err, errors.New("editing interaction response"))
	}

	return raffle.UpdateMessage(s)
}
