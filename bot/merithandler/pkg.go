package merithandler

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

type MeritCommand struct{}

var _ command.ApplicationCommand = (*MeritCommand)(nil)

func New() command.ApplicationCommand {
	return &MeritCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (m *MeritCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (m *MeritCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// CommandHandler implements [command.ApplicationCommand].
func (m *MeritCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("merit command")

	user, err := members.Get(i.Member.User.ID)
	if err != nil {
		return errors.Wrap(err, "getting user from storage for merit command")
	}

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingMember, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		return errors.Wrap(err, "getting receiving member")
	}

	if err := receivingMember.GiveMerit(data.Options[1].StringValue(), user); err != nil {
		return errors.Wrap(err, "giving member merit")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the merit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to merit command")
	}

	return nil
}

// ModalHandler implements [command.ApplicationCommand].
func (m *MeritCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (m *MeritCommand) Name() string {
	return "merit"
}

// OnAfter implements [command.ApplicationCommand].
func (m *MeritCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (m *MeritCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (m *MeritCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (m *MeritCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (m *MeritCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "merit",
		Description: "give a merit to a member",
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
				Description: "why are you giving this member a merit",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	}, nil
}
