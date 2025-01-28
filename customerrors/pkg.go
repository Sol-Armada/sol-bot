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

	// bot hanlder errors
	ChannelNotExist         = e.New("channel does not exist")
	InvalidPermissions      = e.New("invalid permissions")
	InvalidSubcommand       = e.New("invalid subcommand")
	InvalidAutocomplete     = e.New("invalid autocomplete")
	InvalidButton           = e.New("invalid button")
	InvalidAttendanceRecord = e.New("invalid attendance record")
)

func ErrorResponse(s *discordgo.Session, i *discordgo.Interaction, message string, errorCode *string) {
	if message == "" {
		message = "There was an error! Please try again in a fwe minutes or let the @Officers know"
	}
	if errorCode != nil {
		message += "\n\nError code: " + *errorCode
	}
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: message,
		},
	}); err != nil {
		if _, err := s.FollowupMessageCreate(i, false, &discordgo.WebhookParams{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		}); err != nil {
			log.WithError(err).Error("responding to event command interaction")
		}
	}
}

func Is(err, target error) bool {
	return e.Is(err, target)
}

func Wrap(err error, message string) error {
	return e.Join(err, e.New(message))
}
