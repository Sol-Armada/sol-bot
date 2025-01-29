package attendancehandler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/dkp"
)

func stayedSubmitButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	if err := s.ChannelMessageDelete(i.Message.ChannelID, i.Message.ID); err != nil {
		return err
	}

	content := "Tokens has been distributed"

	for _, member := range attendance.Members {
		amount := 10
		if err := dkp.New(member.Id, 10, dkp.Attendance, &attendanceId, nil).Save(); err != nil {
			return err
		}

		if attendance.Successful {
			if err := dkp.New(member.Id, 20, dkp.EventSuccessful, &attendanceId, nil).Save(); err != nil {
				return err
			}
			amount += 20
		}

		if slices.Contains(attendance.Stayed, member.Id) {
			if err := dkp.New(member.Id, 10, dkp.AttendanceFull, &attendanceId, nil).Save(); err != nil {
				return err
			}
			amount += 10
		}

		content += fmt.Sprintf("\n<@%s> has received %d Tokens", member.Id, amount)
	}

	attendanceMessage := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendance.ChannelId,
		ID:         attendance.MessageId,
		Components: &attendanceMessage.Components,
		Embed:      attendanceMessage.Embed,
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}
