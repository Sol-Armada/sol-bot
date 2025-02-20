package rafflehandler

import (
	"context"
	"errors"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func backOut(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle back out button")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Join(err, errors.New("responding to back out button"))
	}

	data := i.MessageComponentData()

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return errors.Join(err, errors.New("getting raffle"))
	}

	if err := raffle.RemoveTicket(i.Member.User.ID).Save(); err != nil {
		return errors.Join(err, errors.New("removing ticket"))
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: "You have successfully backed out of the raffle.",
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return errors.Join(err, errors.New("creating followup message"))
	}

	return raffle.UpdateMessage(s)
}
