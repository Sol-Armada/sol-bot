package handlers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/users"
)

func AttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	storage := users.GetStorage()
	user, err := storage.GetUser(i.Member.User.ID)
	if err != nil {
		panic(err)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%d events", user.Events),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		panic(err)
	}
}
