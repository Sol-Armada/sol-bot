package rafflehandler

import (
	"context"
	"fmt"
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

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return err
	}

	if raffle.AttedanceId != "" {
		attendanceRecord, err := attendance.Get(raffle.AttedanceId)
		if err != nil {
			return err
		}

		if _, ok := attendanceRecord.GetMember(i.Member.User.ID); !ok {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You did not attend this event and cannot enter the raffle.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}

	won, err := raffle.MemberWonLast(i.Member.User.ID)
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
			CustomID: fmt.Sprintf("raffle:add_entries:%s", raffleId),
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
