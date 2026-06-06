package loothandler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

type LootCommand struct {
}

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"name": startAutocomplete,
}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"end":          end,
	"update_entry": updateEntry,
	"exit":         exit,
}

func New() command.ApplicationCommand {
	return &LootCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (l *LootCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loot autocomplete handler")

	data := i.ApplicationCommandData()
	handler, ok := autoCompletes[data.Options[0].Name]
	logger = logger.With(
		slog.String("subcommand", data.Options[0].Name),
	)
	ctx = utils.SetLoggerToContext(ctx, logger)
	if !ok {
		return customerrors.InvalidAutocomplete
	}

	logger.Debug("calling autocomplete handler")

	return handler(ctx, s, i)
}

// ButtonHandler implements [command.ApplicationCommand].
func (l *LootCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loothandler button handler")

	data := i.Interaction.MessageComponentData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := buttons[action]
	if !ok {
		return customerrors.InvalidButton
	}

	return handler(ctx, s, i)
}

// CommandHandler implements [command.ApplicationCommand].
func (l *LootCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("loot command handler")

	return start(ctx, s, i)
}

// ModalHandler implements [command.ApplicationCommand].
func (l *LootCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (l *LootCommand) Name() string {
	return "loot"
}

// OnAfter implements [command.ApplicationCommand].
func (l *LootCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (l *LootCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (l *LootCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (l *LootCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (l *LootCommand) Setup() (*discordgo.ApplicationCommand, error) {
	options := []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "name",
			Description:  "The name of the loot. If you associate with an event, it will use the event name.",
			Required:     true,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "timer",
			Description: "The timer to set for the loot in minutes. d = days, h = hours, m = minutes. Example: 1d2h3m",
			Required:    true,
		},
	}

	for i := range 10 {
		option := &discordgo.ApplicationCommandOption{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         fmt.Sprintf("item-%d", i+1),
			Description:  "The item to add. Use colon to separate the name and amount. Example: item:amount",
			Autocomplete: true,
		}

		if i == 0 {
			option.Required = true
		}

		options = append(options, option)
	}

	return &discordgo.ApplicationCommand{
		Name:        "loot",
		Description: "Need/Greed",
		Type:        discordgo.ChatApplicationCommand,
		Options:     options,
	}, nil
}

// SetupAliases implements [command.ApplicationCommand].
func (l *LootCommand) SetupAliases() ([]*discordgo.ApplicationCommand, error) {
	return nil, nil
}

var _ command.ApplicationCommand = (*LootCommand)(nil)
