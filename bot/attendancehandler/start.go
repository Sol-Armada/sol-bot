package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func startEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	participants, err := a.Participants()
	if err != nil {
		return err
	}

	for _, participant := range participants {
		if participant.JoinedAtStart {
			continue
		}

		participant.JoinedAtStart = true
		if err := participant.Save(a.Id); err != nil {
			return err
		}
	}

	if err := a.SetStatus(attendance.AttendanceStatusActive); err != nil {
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

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{},
	})
}
