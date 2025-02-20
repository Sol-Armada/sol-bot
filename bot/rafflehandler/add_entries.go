package rafflehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/utils"
)

func addEntries(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle buy in button")

	data := i.MessageComponentData()

	raffleId := strings.Split(data.CustomID, ":")[2]

	raffle, err := raffles.Get(raffleId)
	if err != nil {
		return err
	}

	attendanceRecord, err := attendance.Get(raffle.AttedanceId)
	if err != nil {
		return err
	}

	for _, member := range attendanceRecord.Members {
		if member.Id == i.Member.User.ID {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You did not attend this event and cannot enter the raffle.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("raffle:add_entries:%s", raffleId),
			Title:    "Add Entries",
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
