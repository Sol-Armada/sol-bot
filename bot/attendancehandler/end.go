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

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	if a.ChannelId == "" || a.MessageId == "" {
		a.ChannelId = i.ChannelID
		a.MessageId = i.Message.ID
	}

	if err := a.Record(); err != nil {
		return err
	}

	attendanceMessage, err := a.ToDiscordMessage()
	if err != nil {
		return err
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    a.ChannelId,
		ID:         a.MessageId,
		Components: &attendanceMessage.Components,
		Embeds:     &attendanceMessage.Embeds,
	}); err != nil {
		return err
	}

	go directMessageAttendees(s, utils.GetLoggerFromContext(ctx), a)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}
