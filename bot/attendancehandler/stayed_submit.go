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

var stayed map[string][]string = make(map[string][]string)

func stayedSubmitButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	membersWhoStayed := stayed[attendanceId]

	for _, member := range membersWhoStayed {
		// Do something with the member
		_ = member
	}

	if err := s.ChannelMessageDelete(i.Message.ChannelID, i.Message.ID); err != nil {
		return err
	}

	content := "StarCoin has been distributed"

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

		if slices.Contains(membersWhoStayed, member.Id) {
			if err := dkp.New(member.Id, 10, dkp.AttendanceFull, &attendanceId, nil).Save(); err != nil {
				return err
			}
			amount += 10
		}

		content += fmt.Sprintf("\n<@%s> has received %d StarCoins", member.Id, amount)
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}
