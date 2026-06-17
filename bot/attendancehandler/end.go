package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/utils"
)

func endEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	if err := attendance.Record(); err != nil {
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

	go directMessageAttendees(s, utils.GetLoggerFromContext(ctx), attendance)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}
