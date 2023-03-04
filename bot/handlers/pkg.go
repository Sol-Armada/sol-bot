package handlers

import (
	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

func ErrorResponse(s *discordgo.Session, i *discordgo.Interaction, message string) {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: message,
		},
	}); err != nil {
		log.WithError(err).Error("responding to event command interaction")
	}
}
