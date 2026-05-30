package settingshandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

type SettingsCommand struct{}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"dm_opt_out": dmOptOutButtonHandler,
}

func New() command.ApplicationCommand {
	return &SettingsCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (*SettingsCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (*SettingsCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("settings button handler")

	data := i.Interaction.MessageComponentData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := buttons[action]
	if !ok {
		return customerrors.InvalidButton
	}

	return handler(ctx, s, i)
}

// CommandHandler implements [command.ApplicationCommand].
func (*SettingsCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ModalHandler implements [command.ApplicationCommand].
func (*SettingsCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (s *SettingsCommand) Name() string {
	return "settings"
}

// OnAfter implements [command.ApplicationCommand].
func (*SettingsCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (*SettingsCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (*SettingsCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (*SettingsCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (s *SettingsCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return nil, nil
}

// SetupAliases implements [command.ApplicationCommand].
func (s *SettingsCommand) SetupAliases() ([]*discordgo.ApplicationCommand, error) {
	return nil, nil
}

var _ command.ApplicationCommand = (*SettingsCommand)(nil)
