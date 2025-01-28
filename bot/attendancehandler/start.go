package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
)

func startEventButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	recordId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	record, err := attendance.Get(recordId)
	if err != nil {
		return err
	}

	for _, member := range record.Members {
		if member.Id == "" {
			continue
		}

		record.FromStart = append(record.FromStart, member)
	}

	record.Active = true

	if err := record.Save(); err != nil {
		return err
	}

	recordMessage := record.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    record.ChannelId,
		ID:         record.MessageId,
		Components: &recordMessage.Components,
		Embed:      recordMessage.Embed,
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{},
	})
}
