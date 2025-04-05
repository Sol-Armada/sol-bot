package attendancehandler

import (
	"context"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	attnc "github.com/sol-armada/sol-bot/attendance"
)

func startEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attnc.Get(attendanceId)
	if err != nil {
		return err
	}

	for _, member := range attendance.Members {
		if member.Id == "" || slices.Contains(attendance.FromStart, member.Id) {
			continue
		}

		attendance.FromStart = append(attendance.FromStart, member.Id)
	}

	attendance.Active = true
	attendance.Status = attnc.AttendanceStatusActive

	if err := attendance.Save(); err != nil {
		return err
	}

	attendanceMessage := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendance.ChannelId,
		ID:         attendance.MessageId,
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
