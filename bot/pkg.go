package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/bot/attendancehandler"
	"github.com/sol-armada/sol-bot/bot/giveawayhandler"
	"github.com/sol-armada/sol-bot/bot/rafflehandler"
	"github.com/sol-armada/sol-bot/bot/tokenshandler"
	"github.com/sol-armada/sol-bot/bot/wikelohandler"
	"github.com/sol-armada/sol-bot/bot/yourelatehandler"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

type Bot struct {
	GuildId  string
	ClientId string

	logger *slog.Logger
	ctx    context.Context

	*discordgo.Session
}

type Handler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error

var bot *Bot

// command handlers
var commandHandlers = map[string]Handler{
	"help":         helpCommandHandler,
	"attendance":   attendancehandler.CommandHandler,
	"profile":      profileCommandHandler,
	"merit":        giveMeritCommandHandler,
	"demerit":      giveDemeritCommandHandler,
	"rankups":      rankUpsCommandHandler,
	"tokens":       tokenshandler.CommandHandler,
	"raffle":       rafflehandler.CommandHandler,
	"giveaway":     giveawayhandler.CommandHandler,
	"wikelo":       wikelohandler.CommandHandler,
	"you_are_late": yourelatehandler.CommandHandler,
}

var autocompleteHandlers = map[string]Handler{
	"attendance": attendancehandler.AutocompleteHander,
	"tokens":     tokenshandler.AutocompleteHandler,
	"raffle":     rafflehandler.AutocompleteHander,
	"giveaway":   giveawayhandler.AutocompleteHander,
}

var buttonHandlers = map[string]Handler{
	"attendance": attendancehandler.ButtonHandler,
	"tokens":     tokenshandler.ButtonHandler,
	"raffle":     rafflehandler.ButtonHandler,
	"giveaway":   giveawayhandler.ButtonHandler,
}

var modalHandlers = map[string]Handler{
	"raffle": rafflehandler.ModalHandler,
}

var onboardingButtonHanlders = map[string]Handler{
	"choice":   onboardingButtonHandler,
	"tryagain": onboardingTryAgainHandler,
}

var onboardingModalHandlers = map[string]Handler{
	"onboard":   onboardingModalHandler,
	"rsihandle": onboardingTryAgainModalHandler,
}

var attendanceModalHandlers = map[string]Handler{
	"payout": attendancehandler.AddPayoutModalHandler,
}

var validateButtonHandlers = map[string]Handler{
	"start": startValidateButtonHandler,
	"check": checkValidateButtonHandler,
}

func New() (*Bot, error) {
	slog.Info("creating discord bot")
	b, err := discordgo.New(fmt.Sprintf("Bot %s", settings.GetString("DISCORD.BOT_TOKEN")))
	if err != nil {
		return nil, err
	}

	if _, err := b.Guild(settings.GetString("DISCORD.GUILD_ID")); err != nil {
		return nil, err
	}

	b.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates + discordgo.IntentsGuildMessageReactions + discordgo.PermissionAdministrator
	b.Client.Timeout = 5 * time.Second

	bot = &Bot{
		GuildId:  settings.GetString("DISCORD.GUILD_ID"),
		ClientId: settings.GetString("DISCORD.CLIENT_ID"),
		logger:   slog.Default(),
		ctx:      context.Background(),
		Session:  b,
	}

	return bot, nil
}

func GetBot() (*Bot, error) {
	if bot == nil {
		b, err := New()
		if err != nil {
			return nil, err
		}
		bot = b
	}

	return bot, nil
}

func (b *Bot) Setup() error {
	b.logger.Debug("setting up handlers and commands")

	defer func() {
		b.logger.Debug("updating custom status")
		if err := b.UpdateCustomStatus("ready to serve"); err != nil {
			b.logger.Error("failed to update custom status", "error", err)
		}
	}()

	b.logger.Debug("adding ready handler")
	b.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		b.logger.Info("bot is ready", "user", r.User.Username)
	})

	b.logger.Debug("adding interaction handler")

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		member, err := members.Get(i.Member.User.ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				b.logger.Error("getting member for incoming interaction", "error", err)
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Ran into an error, please try again in a few minutes",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			member = members.New(i.Member)
		}

		ctx := utils.SetMemberToContext(b.ctx, member)

		logger := b.logger.With(
			"guild", b.GuildId,
			"user", i.Member.User.ID,
		)

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			logger = logger.With(
				"interaction_type", "application command",
				"name", i.ApplicationCommandData().Name,
			)
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				ctx = utils.SetLoggerToContext(ctx, logger.With("interaction_type", "application command"))
				err = h(ctx, s, i)
			}
		case discordgo.InteractionApplicationCommandAutocomplete:
			parentCommandData := i.ApplicationCommandData()

			logger = logger.With(
				"interaction_type", "application command autocomplete",
				"command", parentCommandData.Name,
			)

			if handler, ok := autocompleteHandlers[parentCommandData.Name]; ok {
				ctx = utils.SetLoggerToContext(ctx, logger.With(
					"command", parentCommandData.Options[0].Name,
				))
				err = handler(ctx, s, i)
			}
		case discordgo.InteractionMessageComponent:
			commandSplit := strings.Split(i.MessageComponentData().CustomID, ":")
			command := commandSplit[0]
			subcommand := commandSplit[1]

			logger = logger.With(
				"interaction_type", "message command",
				"custom_id", i.MessageComponentData().CustomID,
				"button_command", command,
			)

			ctx = utils.SetLoggerToContext(ctx, logger)
			switch command {
			case "onboarding":
				if h, ok := onboardingButtonHanlders[subcommand]; ok {
					err = h(ctx, s, i)
				}
			case "validate":
				if h, ok := validateButtonHandlers[subcommand]; ok {
					err = h(ctx, s, i)
				}
			default:
				err = buttonHandlers[command](ctx, s, i)
			}

		case discordgo.InteractionModalSubmit:
			logger = logger.With(
				"interaction_type", "modal submit",
			)
			ctx = utils.SetLoggerToContext(ctx, logger)
			command := strings.Split(i.ModalSubmitData().CustomID, ":")
			subCommand := command[1]
			switch command[0] {
			case "onboarding":
				if h, ok := onboardingModalHandlers[subCommand]; ok {
					err = h(ctx, s, i)
				}
			case "attendance":
				if h, ok := attendanceModalHandlers[subCommand]; ok {
					err = h(ctx, s, i)
				}
			default:
				err = modalHandlers[command[0]](ctx, s, i)
			}
		}

		if err != nil { // handle any errors returned
			msg := "It looks like I ran into an error. I have logged it and someone will look into it. Ask an @Officer if you need help"

			switch err {
			case InvalidSubcommand:
				msg = "Invalid subcommand"
			case InvalidPermissions, customerrors.InvalidPermissions:
				msg = "You don't have permission to do that"
			case InvalidAttendanceRecord:
				msg = "That is not a valid attendance record"
			default:
				if settings.GetString("DISCORD.ERROR_CHANNEL_ID") != "" {
					inputValues := ""
					if len(i.ApplicationCommandData().Options) > 0 {
						for _, option := range i.ApplicationCommandData().Options {
							inputValues += fmt.Sprintf("%s: %v\n", option.Name, option.Value)
						}
					}

					_, _ = b.ChannelMessageSendComplex(settings.GetString("DISCORD.ERROR_CHANNEL_ID"), &discordgo.MessageSend{
						Content: "Ran into an error",
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Error",
								Description: err.Error(),
								Fields: []*discordgo.MessageEmbedField{
									{Name: "Who ran the command", Value: i.Member.User.Mention(), Inline: false},
									{Name: "Command", Value: i.ApplicationCommandData().Name, Inline: true},
									{Name: "Values", Value: inputValues, Inline: true},
									{Name: "Error", Value: err.Error(), Inline: false},
								},
								Timestamp: time.Now().Format("2006-01-02 15:04:05 -0700 MST"),
							},
						},
					})
				}
			}

			switch i.Interaction.Type {
			case discordgo.InteractionApplicationCommand:
				logger.Error("running command",
					"command_data", i.ApplicationCommandData(),
					"error", err)
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}); err != nil {
					logger.Warn("creating interaction response", "error", err)
					if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					}); err != nil {
						logger.Error("creating followup message", "error", err)
					}
				}
			case discordgo.InteractionMessageComponent:
				logger.Error("running command",
					"component_data", i.MessageComponentData(),
					"error", err)
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}); err != nil {
					logger.Warn("creating component interaction response", "error", err)
					if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					}); err != nil {
						logger.Error("creating component followup message", "error", err)
					}
				}
			case discordgo.InteractionModalSubmit:
				logger.Error("running command",
					"modal_data", i.ModalSubmitData(),
					"error", err)
				if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				}); err != nil {
					logger.Error("creating component followup message", "error", err)
				}
			default:
				logger.Error("unknown interaction type", "interaction_type", i.Interaction.Type)
			}
		}
	})

	// onboarding
	if settings.GetBool("FEATURES.ONBOARDING.ENABLE") {
		// watch for on join and leave
		b.AddHandler(onJoinHandler)
		b.AddHandler(onLeaveHandler)

		if err := setupOnboarding(); err != nil {
			return errors.Wrap(err, "setting up onboarding")
		}
	}

	// register commands

	// misc commands
	slog.Debug("creating profile command")
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "View your profile",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "member",
				Description: "View a member's profile (Officer only)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "force_update",
				Description: "Force update profile",
				Required:    false,
			},
		},
	}); err != nil {
		return errors.Wrap(err, "creating profile command")
	}

	slog.Debug("creating help command")

	helpCmd, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "View help",
	})
	if err != nil {
		slog.Error("failed to create help command", "error", err)
		return errors.Wrap(err, "creating help command")
	}
	slog.Debug("help command created successfully", "command_id", helpCmd.ID)

	// rank up
	slog.Debug("creating rankup command")

	rankupCmd, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "rankups",
		Description: "Rank up your RSI profile",
	})
	if err != nil {
		slog.Error("failed to create rankup command", "error", err)
		return errors.Wrap(err, "creating rankup command")
	}
	slog.Debug("rankup command created successfully", "command_id", rankupCmd.ID)

	// attendance
	if settings.GetBool("FEATURES.ATTENDANCE.ENABLE") {
		slog.Debug("using attendance feature")

		cmd, err := attendancehandler.Setup()
		if err != nil {
			return errors.Wrap(err, "setting attendance commands")
		}

		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
			return errors.Wrap(err, "creating attendance command")
		}
	}

	// merit
	if settings.GetBool("FEATURES.MERIT.ENABLE") {
		slog.Debug("using merit feature")
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
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
		}); err != nil {
			return errors.Wrap(err, "failed creating merit command")
		}
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
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
		}); err != nil {
			return errors.Wrap(err, "failed creating demerit command")
		}
	}

	// activity tracking
	if settings.GetBool("FEATURES.ACTIVITY_TRACKING.ENABLE") {
		b.AddHandler(onVoiceUpdate)
	}

	// tokens
	if settings.GetBool("FEATURES.TOKENS.ENABLE") {
		slog.Debug("using tokens feature")
		cmd, err := tokenshandler.Setup()
		if err != nil {
			return errors.Wrap(err, "setting tokens commands")
		}

		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
			return errors.Wrap(err, "creating tokens command")
		}
	}

	// raffles
	if settings.GetBool("FEATURES.RAFFLES.ENABLE") {
		slog.Debug("using raffles feature")
		cmd, err := rafflehandler.Setup()
		if err != nil {
			return errors.Wrap(err, "setting raffles commands")
		}

		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
			return errors.Wrap(err, "creating raffles command")
		}
	}

	// giveaways
	if settings.GetBool("FEATURES.GIVEAWAYS.ENABLE") {
		slog.Debug("using giveaways feature")
		if err := giveaway.Load(b.Session); err != nil {
			return errors.Wrap(err, "loading giveaways")
		}

		cmd, err := giveawayhandler.Setup()
		if err != nil {
			return errors.Wrap(err, "setting giveaways commands")
		}
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
			return errors.Wrap(err, "creating giveaways command")
		}
	}

	// b.AddHandler(OnRoleChange)

	cmd, err := wikelohandler.Setup()
	if err != nil {
		return errors.Wrap(err, "setting up wikelohandler")
	}
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
		return errors.Wrap(err, "creating wikelohandler command")
	}

	cmd, err = yourelatehandler.Setup()
	if err != nil {
		return errors.Wrap(err, "setting up yourelatehandler")
	}
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmd); err != nil {
		return errors.Wrap(err, "creating yourelatehandler command")
	}

	b.logger.Debug("all handlers and commands registered - opening Discord connection")

	if err := b.Open(); err != nil {
		b.logger.Error("failed to open Discord connection", "error", err)
		return errors.Wrap(err, "opening Discord connection")
	}

	b.logger.Debug("Discord connection opened successfully")
	return nil
}

func (b *Bot) Close() error {
	b.logger.Info("stopping bot")

	clearStatusMessages()
	if err := b.UpdateCustomStatus("shutting down"); err != nil {
		b.logger.Error("failed to update custom status", "error", err)
	}

	// clear commands
	cmds, err := b.ApplicationCommands(b.ClientId, b.GuildId)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		b.logger.Debug("deleting command", "command", cmd.Name)
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, cmd.ID); err != nil {
			return err
		}
	}

	return b.Session.Close()
}

func (b *Bot) UpdateCustomStatus(status string) error {
	if status == "" {
		status = "ready to serve"
	}
	return b.Session.UpdateCustomStatus(status)
}

func allowed(discordMember *discordgo.Member, feature string) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
