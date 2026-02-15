package giveawayhandler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/utils"
)

type GiveawayCommand struct{}

var _ command.ApplicationCommand = (*GiveawayCommand)(nil)

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"name": startAutocomplete,
}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"end":          end,
	"update_entry": updateEntry,
	"view_entries": viewEntries,
	"exit":         exit,
}

var modals = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{}

func New() command.ApplicationCommand {
	return &GiveawayCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (g *GiveawayCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway autocomplete handler")

	if !utils.Allowed(i.Member, "GIVEAWAYS") {
		return customerrors.InvalidPermissions
	}

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
func (g *GiveawayCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway button handler")

	if ok := checkExists(ctx, i); !ok {
		customerrors.ErrorResponse(s, i.Interaction, "That giveaway no longer exists in the system! I will remove it so that doesn't happen again", nil)
		return s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
	}

	data := i.Interaction.MessageComponentData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := buttons[action]
	if !ok {
		return customerrors.InvalidButton
	}

	return handler(ctx, s, i)
}

// CommandHandler implements [command.ApplicationCommand].
func (g *GiveawayCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway command handler")

	if !utils.Allowed(i.Member, "GIVEAWAYS") {
		return customerrors.InvalidPermissions
	}

	return start(ctx, s, i)
}

// ModalHandler implements [command.ApplicationCommand].
func (g *GiveawayCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway modal handler")

	if ok := checkExists(ctx, i); !ok {
		customerrors.ErrorResponse(s, i.Interaction, "That giveaway no longer exists in the system! I will remove it so that doesn't happen again", nil)
		return s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
	}

	data := i.Interaction.ModalSubmitData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := modals[action]
	if !ok {
		return customerrors.InvalidModal
	}

	return handler(ctx, s, i)
}

// Name implements [command.ApplicationCommand].
func (g *GiveawayCommand) Name() string {
	return "giveaway"
}

// OnAfter implements [command.ApplicationCommand].
func (g *GiveawayCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	panic("unimplemented")
}

// OnBefore implements [command.ApplicationCommand].
func (g *GiveawayCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	panic("unimplemented")
}

// OnError implements [command.ApplicationCommand].
func (g *GiveawayCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	panic("unimplemented")
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (g *GiveawayCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	panic("unimplemented")
}

// Setup implements [command.ApplicationCommand].
func (g *GiveawayCommand) Setup() (*discordgo.ApplicationCommand, error) {
	options := []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "name",
			Description:  "The name of the giveaway. If you associate with an event, it will use the event name.",
			Required:     true,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "timer",
			Description: "The timer to set for the giveaway in minutes. d = days, h = hours, m = minutes. Example: 1d2h3m",
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
		Name:        "giveaway",
		Description: "Start a giveaway",
		Type:        discordgo.ChatApplicationCommand,
		Options:     options,
	}, nil
}

func checkExists(ctx context.Context, i *discordgo.InteractionCreate) bool {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("giveaway check exists")

	data := i.MessageComponentData()
	giveawayId := strings.Split(data.CustomID, ":")[2]

	return giveaway.GetGiveaway(giveawayId) != nil
}
