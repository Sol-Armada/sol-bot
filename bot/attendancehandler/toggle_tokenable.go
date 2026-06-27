package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func toggleTokenableButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	if a.ChannelId == "" || a.MessageId == "" {
		a.ChannelId = i.ChannelID
		a.MessageId = i.Message.ID
	}

	a.Tokenable = !a.Tokenable
	if !a.Tokenable {
		a.Successful = false
	}
	if err := a.Save(); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}
