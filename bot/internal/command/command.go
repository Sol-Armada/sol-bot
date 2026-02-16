package command

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Name     string                                               `json:"name"`
	When     time.Time                                            `json:"when"`
	User     string                                               `json:"user"`
	Options  []*discordgo.ApplicationCommandInteractionDataOption `json:"options"`
	Type     discordgo.InteractionType                            `json:"type"`
	ButtonId string                                               `json:"button_id,omitempty"`
	Error    string                                               `json:"error,omitempty"`
}

type ApplicationCommand interface {
	Name() string
	Setup() (*discordgo.ApplicationCommand, error)

	OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error)

	CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

func RunCommand(ctx context.Context, cmd ApplicationCommand, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	switch i.Type {
	case discordgo.InteractionApplicationCommandAutocomplete, discordgo.InteractionMessageComponent, discordgo.InteractionModalSubmit:
	default:
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return err
		}
	}

	if err := cmd.OnBefore(ctx, s, i); err != nil {
		cmd.OnError(ctx, s, i, err)
		return err
	}

	var err error
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		err = cmd.CommandHandler(ctx, s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		err = cmd.AutocompleteHandler(ctx, s, i)
	case discordgo.InteractionMessageComponent:
		err = cmd.ButtonHandler(ctx, s, i)
	case discordgo.InteractionModalSubmit:
		err = cmd.ModalHandler(ctx, s, i)
	}
	if err != nil {
		cmd.OnError(ctx, s, i, err)
		return err
	}

	if err := cmd.OnAfter(ctx, s, i); err != nil {
		cmd.OnError(ctx, s, i, err)
		return err
	}
	return nil
}
