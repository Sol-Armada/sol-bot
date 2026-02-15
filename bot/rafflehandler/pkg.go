package rafflehandler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

type RaffleCommand struct{}

var _ command.ApplicationCommand = (*RaffleCommand)(nil)

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"name": startAutocomplete,
}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"back_out":    backOut,
	"add_entries": addEntries,
	"end":         end,
	"cancel":      cancel,
	"entries":     entries,
}

var modals = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"add_entries": addEntriesModal,
}

func New() command.ApplicationCommand {
	return &RaffleCommand{}
}

// Setup implements [command.ApplicationCommand].
func (r *RaffleCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "raffle",
		Description: "Start a raffle",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "name",
				Description:  "The name of the raffle. If linked to an attendance record, use the attendance record Name",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "prize",
				Description:  "The prize. To add quantity, use 'item:qty'. E.g. 'Pickles:2'",
				Required:     true,
				Autocomplete: false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "test",
				Description: "Whether this is a test raffle (won't be logged)",
				Required:    false,
			},
		},
	}, nil
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (r *RaffleCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle autocomplete handler")

	// if !utils.Allowed(i.Member, "RAFFLES") {
	// 	return customerrors.InvalidPermissions
	// }

	data := i.ApplicationCommandData()
	optionName := data.Options[0].Name

	handler, ok := autoCompletes[optionName]
	if !ok {
		return customerrors.InvalidAutocomplete
	}

	logger = logger.With(
		slog.String("subcommand", optionName),
	)
	ctx = utils.SetLoggerToContext(ctx, logger)

	logger.Debug("calling autocomplete handler")

	return handler(ctx, s, i)
}

// ButtonHandler implements [command.ApplicationCommand].
func (r *RaffleCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {

	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle button handler")

	data := i.Interaction.MessageComponentData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := buttons[action]
	if !ok {
		return customerrors.InvalidButton
	}

	return handler(ctx, s, i)
}

// CommandHandler implements [command.ApplicationCommand].
func (r *RaffleCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle command handler")

	// if !utils.Allowed(i.Member, "RAFFLES") {
	// 	return customerrors.InvalidPermissions
	// }

	return start(ctx, s, i)
}

// ModalHandler implements [command.ApplicationCommand].
func (r *RaffleCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("raffle modal handler")

	data := i.Interaction.ModalSubmitData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := modals[action]
	if !ok {
		return customerrors.InvalidModal
	}

	return handler(ctx, s, i)
}

// Name implements [command.ApplicationCommand].
func (r *RaffleCommand) Name() string {
	return "raffle"
}

// OnAfter implements [command.ApplicationCommand].
func (r *RaffleCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (r *RaffleCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (r *RaffleCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (r *RaffleCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}
