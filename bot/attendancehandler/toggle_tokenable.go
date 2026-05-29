package attendancehandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func toggleTokenableButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := i.MessageComponentData().CustomID

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	attendance.Tokenable = !attendance.Tokenable
	if err := attendance.Save(); err != nil {
		return err
	}

	attendanceMessage, err := attendance.ToDiscordMessage()
	if err != nil {
		return err
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendance.ChannelId,
		ID:         attendance.MessageId,
		Components: &attendanceMessage.Components,
		Embeds:     &attendanceMessage.Embeds,
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
	})
}
