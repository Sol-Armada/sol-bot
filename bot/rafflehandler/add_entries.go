package rafflehandler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
)

func addEntries(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle buy in button")

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

	if raffle.AttendanceId != "" {
		attendanceRecord, err := attendance.Get(raffle.AttendanceId)
		if err != nil {
			return err
		}

		participated, err := attendanceRecord.HasParticipant(i.Member.User.ID)
		if err != nil {
			return err
		}

		if !participated {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You did not attend this event and cannot enter the raffle.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}

	won, err := raffle.MemberWonLast(ctx, i.Member.User.ID)
	if err != nil {
		return err
	}

	if won {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are on a raffle lockout and cannot enter this one. You will be able to enter the next.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	tokens, err := tokens.GetBalanceByMemberId(i.Member.User.ID)
	if err != nil {
		return err
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("raffle:add_entries:%d", raffleId),
			Title:    fmt.Sprintf("Add Entries (%d max)", tokens),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "ticket_count",
							Label:       "Ticket Count (1 Token = 1 Ticket)",
							Placeholder: "Enter the number of tokens you want to use",
							Style:       discordgo.TextInputShort,
						},
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func addEntriesModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle buy in modal")

	data := i.ModalSubmitData()

	raffleIdStr := strings.Split(data.CustomID, ":")[2]

	raffleId, err := strconv.Atoi(raffleIdStr)
	if err != nil {
		return err
	}

	raffle, err := raffles.Get(int32(raffleId))
	if err != nil {
		return err
	}

	ticketCountRaw := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	ticketCount, err := strconv.Atoi(ticketCountRaw)
	if err != nil {
		// return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Content: "The amount must be a number. Please try again.",
		// 		Flags:   discordgo.MessageFlagsEphemeral,
		// 	},
		// })

		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: new("The amount must be a number. Please try again."),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	logger = logger.With(
		slog.Int64("raffle_id", int64(raffleId)),
		slog.Int("ticket_count", ticketCount),
		slog.String("member_id", i.Member.User.ID),
	)
	logger.Debug("raffle buy in modal submit")

	if ticketCount < 1 {
		// return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Content: "You must buy at least one ticket. Please try again.",
		// 		Flags:   discordgo.MessageFlagsEphemeral,
		// 	},
		// })
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: new("You must buy at least one ticket. Please try again."),
			Flags:   discordgo.MessageFlagsEphemeral,
		}); err != nil {
			return err
		}
	}

	balance, err := tokens.GetBalanceByMemberId(i.Member.User.ID)
	if err != nil {
		return err
	}

	if balance < ticketCount {
		logger.Debug("insufficient balance", "balance", balance)

		// return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Content: fmt.Sprintf("You only have %d Tokens! Please try again.", balance),
		// 		Flags:   discordgo.MessageFlagsEphemeral,
		// 	},
		// })
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: new(fmt.Sprintf("You only have %d Tokens! Please try again with a value less than or equal to %d.", balance, balance)),
			Flags:   discordgo.MessageFlagsEphemeral,
		}); err != nil {
			return err
		}
	}

	if err := raffle.AddTicket(i.Member.User.ID, ticketCount).Save(ctx); err != nil {
		return err
	}

	if err := raffle.UpdateMessage(s); err != nil {
		return err
	}

	// return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Content: fmt.Sprintf("%d Tickets submitted!\nIf you want to back out, press the 'Back Out' button. If you want to adjust the amount of tickets, press the 'Edit Entries' button again.\nEditing your entries does not add to, but replaces your previous amount.", ticketCount),
	// 		Flags:   discordgo.MessageFlagsEphemeral,
	// 	},
	// })

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new(fmt.Sprintf("%d Tickets submitted!\nIf you want to back out, press the 'Back Out' button. If you want to adjust the amount of tickets, press the 'Edit Entries' button again.\nEditing your entries does not add to, but replaces your previous amount.", ticketCount)),
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return err
	}

	return nil
}
