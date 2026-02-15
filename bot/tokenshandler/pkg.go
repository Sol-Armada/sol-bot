package tokenshandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

type TokensCommand struct{}

var _ command.ApplicationCommand = (*TokensCommand)(nil)

var subCommands = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"give": giveCommandHandler,
	"take": takeCommandHandler,
}

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"give": reasonAutocompleteHandler,
	"take": reasonAutocompleteHandler,
}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{}

func New() command.ApplicationCommand {
	return &TokensCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (t *TokensCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("tokens autocomplete handler")

	data := i.ApplicationCommandData()

	if handler, ok := autoCompletes[data.Options[0].Name]; ok {
		return handler(ctx, s, i)
	}

	return customerrors.InvalidAutocomplete
}

// ButtonHandler implements [command.ApplicationCommand].
func (t *TokensCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("tokens button handler")

	command := strings.Split(i.MessageComponentData().CustomID, ":")[1]

	if handler, ok := buttons[command]; ok {
		return handler(ctx, s, i)
	}

	return customerrors.InvalidButton
}

// CommandHandler implements [command.ApplicationCommand].
func (t *TokensCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("tokens command handler")

	if !utils.Allowed(i.Member, "TOKENS") {
		return customerrors.InvalidPermissions
	}

	data := i.ApplicationCommandData()

	if handler, ok := subCommands[data.Options[0].Name]; ok {
		return handler(ctx, s, i)
	}

	return customerrors.InvalidSubcommand
}

// ModalHandler implements [command.ApplicationCommand].
func (t *TokensCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (t *TokensCommand) Name() string {
	return "tokens"
}

// OnAfter implements [command.ApplicationCommand].
func (t *TokensCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (t *TokensCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (t *TokensCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (t *TokensCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (t *TokensCommand) Setup() (*discordgo.ApplicationCommand, error) {
	tokensCommandOptions := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "give",
			Description: "Give tokens to a member",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "member",
					Description:  "The member",
					Type:         discordgo.ApplicationCommandOptionUser,
					Required:     true,
					Autocomplete: true,
				},
				{
					Name:        "amount",
					Description: "The amount of tokens",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
				},
				{
					Name:         "reason",
					Description:  "The reason for giving tokens",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
				},
				{
					Name:        "comment",
					Description: "Comment about giving this token",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "take",
			Description: "Take tokens from a member",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "member",
					Description:  "The member",
					Type:         discordgo.ApplicationCommandOptionUser,
					Required:     true,
					Autocomplete: true,
				},
				{
					Name:        "amount",
					Description: "The amount of tokens",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
				},
				{
					Name:        "comment",
					Description: "Comment about taking this token",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}

	return &discordgo.ApplicationCommand{
		Name:        "tokens",
		Description: "Manage tokens",
		Type:        discordgo.ChatApplicationCommand,
		Options:     tokensCommandOptions,
	}, nil
}
