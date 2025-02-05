package attendancehandler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/tokens"
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
		if err := tokens.New(member.Id, 10, tokens.ReasonAttendance, nil, &attendanceId, nil).Save(); err != nil {
			return err
		}

		if attendance.Successful {
			if err := tokens.New(member.Id, 20, tokens.ReasonEventSuccessful, nil, &attendanceId, nil).Save(); err != nil {
				return err
			}
			amount += 20
		}

		if slices.Contains(attendance.Stayed, member.Id) {
			if err := tokens.New(member.Id, 10, tokens.ReasonAttendanceFull, nil, &attendanceId, nil).Save(); err != nil {
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
		Embeds:     &attendanceMessage.Embeds,
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
