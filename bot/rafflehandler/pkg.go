package rafflehandler

import (
	"context"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/utils"
)

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"event": startAutocomplete,
}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"back_out":    backOut,
	"add_entries": addEntries,
	"end":         end,
	"cancel":      cancel,
}

var modals = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"add_entries": addEntriesModal,
}

func Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "raffle",
		Description: "Start a raffle",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "event",
				Description:  "The event to associate",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prize",
				Description: "The prize",
				Required:    true,
			},
		},
	}, nil
}

func CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle command handler")

	if !utils.Allowed(i.Member, "RAFFLES") {
		return customerrors.InvalidPermissions
	}

	return start(ctx, s, i)
}

func AutocompleteHander(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle autocomplete handler")

	if !utils.Allowed(i.Member, "RAFFLES") {
		return customerrors.InvalidPermissions
	}

	data := i.ApplicationCommandData()
	handler, ok := autoCompletes[data.Options[0].Name]
	logger = logger.WithFields(log.Fields{
		"subcommand": data.Options[0].Name,
	})
	ctx = utils.SetLoggerToContext(ctx, logger)
	if !ok {
		return customerrors.InvalidAutocomplete
	}

	logger.Debug("calling autocomplete handler")

	return handler(ctx, s, i)
}

func ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle button handler")

	data := i.Interaction.MessageComponentData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := buttons[action]
	if !ok {
		return customerrors.InvalidButton
	}

	return handler(ctx, s, i)
}

func ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("raffle modal handler")

	data := i.Interaction.ModalSubmitData()
	action := strings.Split(data.CustomID, ":")[1]
	handler, ok := modals[action]
	if !ok {
		return customerrors.InvalidModal
	}

	return handler(ctx, s, i)
}
