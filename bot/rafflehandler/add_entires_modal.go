package rafflehandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
)

func addEntriesModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle buy in modal")

	data := i.ModalSubmitData()

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return err
	}

	ticketCountRaw := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	ticketCount, err := strconv.Atoi(ticketCountRaw)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "The amount must be a number. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	logger = logger.WithFields(
		log.Fields{
			"raffle_id":    raffleId,
			"ticket_count": ticketCount,
			"member_id":    i.Member.User.ID,
		},
	)
	logger.Debug("raffle buy in modal submit")

	if ticketCount < 1 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must buy at least one ticket. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	balance, err := tokens.GetBalanceByMemberId(i.Member.User.ID)
	if err != nil {
		return err
	}

	if balance < ticketCount {
		logger.WithField("balance", balance).Debug("insufficient balance")

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("You only have %d Tokens! Please try again.", balance),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if err := raffle.AddTicket(i.Member.User.ID, ticketCount).Save(); err != nil {
		return err
	}

	if err := raffle.UpdateMessage(s); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%d Tickets submitted!\nIf you want to back out, press the 'Back Out' button. If you want to adjust the amount of tickets, press the 'Add Entries' button again.", ticketCount),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
