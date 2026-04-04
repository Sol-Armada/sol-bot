package blueprinthandler

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/utils"
)

type BlueprintCommand struct{}

var subCommands = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"add":    addHandler,
	"list":   listHandler,
	"remove": removeHandler,
}

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{
	"add":    addAutocompleteHandler,
	"remove": removeAutocompleteHandler,
	"list":   addAutocompleteHandler,
}

var _ command.ApplicationCommand = (*BlueprintCommand)(nil)

func New() command.ApplicationCommand {
	return &BlueprintCommand{}
}

func (c *BlueprintCommand) Name() string {
	return "blueprint"
}

func (c *BlueprintCommand) Setup() (*discordgo.ApplicationCommand, error) {
	subCommands := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "add",
			Description: "Add a blueprint to your profile",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "blueprint",
					Description:  "The ID of the blueprint to add",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "List members with a specific blueprint",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "blueprint",
					Description:  "The ID of the blueprint to search for",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "remove",
			Description: "Remove a blueprint from your profile",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "blueprint",
					Description:  "The ID of the blueprint to remove",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	}

	return &discordgo.ApplicationCommand{
		Name:        c.Name(),
		Description: "Manage blueprints",
		Type:        discordgo.ChatApplicationCommand,
		Options:     subCommands,
	}, nil
}

func (c *BlueprintCommand) SetupAliases() ([]*discordgo.ApplicationCommand, error) {
	cmd, err := c.Setup()
	if err != nil {
		return nil, err
	}
	cmd.Name = "bp"
	return []*discordgo.ApplicationCommand{
		cmd,
	}, nil
}

func (c *BlueprintCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("blueprint autocomplete handler")

	data := i.ApplicationCommandData()
	handler, ok := autoCompletes[data.Options[0].Name]
	logger = logger.With(
		slog.String("subcommand", data.Options[0].Name),
	)
	if !ok {
		logger.Warn("no autocomplete handler found for subcommand")
		return nil
	}
	ctx = utils.SetLoggerToContext(ctx, logger)

	logger.Debug("calling autocomplete handler")

	return handler(ctx, s, i)
}

func (c *BlueprintCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

func (c *BlueprintCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("blueprint command handler")

	data := i.ApplicationCommandData()
	handler, ok := subCommands[data.Options[0].Name]
	logger = logger.With(
		slog.String("subcommand", data.Options[0].Name),
	)
	if !ok {
		logger.Warn("no command handler found for subcommand")
		return nil
	}
	ctx = utils.SetLoggerToContext(ctx, logger)

	logger.Debug("calling command handler")

	return handler(ctx, s, i)
}

func (c *BlueprintCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

func (c *BlueprintCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

func (c *BlueprintCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

func (c *BlueprintCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	if i.Type == discordgo.InteractionMessageComponent {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I ran into an error! You didn't do anything wrong. The right people have been notified and will look into it as soon as possible.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func (c *BlueprintCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}
