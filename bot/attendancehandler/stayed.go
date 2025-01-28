package attendancehandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func stayedSelectHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	recordId := strings.Split(i.MessageComponentData().CustomID, ":")[2]
	_ = recordId
	members := i.MessageComponentData().Values

	stayed[recordId] = members

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{},
	})
}
