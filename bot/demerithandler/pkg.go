package demerithandler

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

type DemeritCommand struct{}

var _ command.ApplicationCommand = (*DemeritCommand)(nil)

func New() command.ApplicationCommand {
	return &DemeritCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (d *DemeritCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (d *DemeritCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// CommandHandler implements [command.ApplicationCommand].
func (d *DemeritCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("demerit command")

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingMember, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		return errors.Wrap(err, "getting receiving member")
	}

	givingMember, err := members.Get(i.Member.User.ID)
	if err != nil {
		return errors.Wrap(err, "getting member from storage for demerit command")
	}

	if err := receivingMember.GiveDemerit(data.Options[1].StringValue(), givingMember); err != nil {
		return errors.Wrap(err, "giving member demerit")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the demerit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to demerit command")
	}

	return nil
}

// ModalHandler implements [command.ApplicationCommand].
func (d *DemeritCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (d *DemeritCommand) Name() string {
	return "demerit"
}

// OnAfter implements [command.ApplicationCommand].
func (d *DemeritCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (d *DemeritCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (d *DemeritCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (d *DemeritCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (d *DemeritCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "demerit",
		Description: "give a demerit to a member",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "member",
				Description:  "who to give the merit to",
				Type:         discordgo.ApplicationCommandOptionUser,
				Required:     true,
				Autocomplete: true,
			},
			{
				Name:        "reason",
				Description: "why are you giving this member a demerit",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	}, nil
}
