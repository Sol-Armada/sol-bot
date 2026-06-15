package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func toggleTokenableButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	attendance.Tokenable = !attendance.Tokenable
	if !attendance.Tokenable {
		attendance.Successful = false
	}
	if err := attendance.Save(); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}
