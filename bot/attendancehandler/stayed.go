package attendancehandler

import (
	"context"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func stayedSelectHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	attendanceId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	a, err := attendance.Get(attendanceId)
	if err != nil {
		return err
	}

	participants, err := a.Participants()
	if err != nil {
		return err
	}

	stayed := i.MessageComponentData().Values

	for _, participant := range participants {
		if !slices.Contains(stayed, participant.Member.Id) {
			participant.StayedUntilEnd = false
		}
		if err := participant.Save(a.Id); err != nil {
			return err
		}
	}

	message, err := a.ToDiscordMessage()
	if err != nil {
		return err
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    a.ChannelId,
		ID:         a.MessageId,
		Components: &message.Components,
		Embeds:     &message.Embeds,
	}); err != nil {
		return err
	}

	return s.InteractionResponseDelete(i.Interaction)
}
