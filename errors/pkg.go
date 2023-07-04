package customerrors

import (
	e "errors"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
)

var (
	ErrMissingName     = e.New("Missing Name")
	ErrMissingStart    = e.New("Missing Start")
	ErrMissingDuration = e.New("Missing Duration")
	ErrMissingId       = e.New("Missing Id")

	ErrStartWrongFormat = e.New("Start is in wrong format")
)

func ErrorResponse(s *discordgo.Session, i *discordgo.Interaction, message string) {
	if message == "" {
		message = "There was an error! Please try again in a fwe minutes or let the @Officers know"
	}
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
