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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

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

	winner, err := raffle.PickWinner()
	if err != nil {
		return err
	}

	if _, err := s.ChannelMessageSend(i.Interaction.ChannelID, fmt.Sprintf("🎊 Congratulations to <@%s>! They have won the raffle! 🎊", winner.Id)); err != nil {
		return err
	}

	if err := tokens.New(winner.Id, raffle.Tickets[winner.Id]*-1, tokens.ReasonWonRaffle, nil, &raffle.AttedanceId, nil).Save(); err != nil {
		return err
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: "The raffle has ended.",
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return err
	}

	return raffle.UpdateMessage(s)
}
