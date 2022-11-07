package handlers

import (
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/users"
)

func AttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	storage := users.GetStorage()
	user, err := storage.GetUser(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user from storage")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%d events", user.Events),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.WithError(err).Error("responding to attendance command interaction")
	}
}
