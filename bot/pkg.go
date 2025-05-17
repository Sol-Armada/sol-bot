package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/bot/attendancehandler"
	"github.com/sol-armada/sol-bot/bot/giveawayhandler"
	"github.com/sol-armada/sol-bot/bot/rafflehandler"
	"github.com/sol-armada/sol-bot/bot/tokenshandler"
	"github.com/sol-armada/sol-bot/bot/wikelohandler"
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
	"help":       helpCommandHandler,
	"attendance": attendancehandler.CommandHandler,
	"profile":    profileCommandHandler,
	"merit":      giveMeritCommandHandler,
	"demerit":    giveDemeritCommandHandler,
	"rankups":    rankUpsCommandHandler,
	"tokens":     tokenshandler.CommandHandler,
	"raffle":     rafflehandler.CommandHandler,
	"giveaway":   giveawayhandler.CommandHandler,
	"wikelo":     wikelohandler.CommandHandler,
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
	log.Info("creating discord bot")
	b, err := discordgo.New(fmt.Sprintf("Bot %s", settings.GetString("DISCORD.BOT_TOKEN")))
	if err != nil {
		return nil, err
	}

	if _, err := b.Guild(settings.GetString("DISCORD.GUILD_ID")); err != nil {
		return nil, err
	}

	b.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates + discordgo.IntentsGuildMessageReactions + discordgo.PermissionAdministrator
	// b.Identify.Intents = discordgo.PermissionAdministrator
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
	b.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		log.WithField("user", r.User.Username).Info("bot is ready")
	})

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		member, err := members.Get(i.Member.User.ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				log.WithError(err).Error("getting member for incomming interaction")
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

		logger := log.WithFields(log.Fields{
			"guild": b.GuildId,
			"user":  i.Member.User.ID,
		})

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			logger = logger.WithFields(log.Fields{
				"interaction_type": "application command",
				"name":             i.ApplicationCommandData().Name,
			})
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				ctx = utils.SetLoggerToContext(ctx, logger.WithField("interaction_type", "application command"))
				err = h(ctx, s, i)
			}
		case discordgo.InteractionApplicationCommandAutocomplete:
			parentCommandData := i.ApplicationCommandData()

			logger = logger.WithFields(log.Fields{
				"interaction_type": "application command autocomplete",
				"command":          parentCommandData.Name,
			})

			if handler, ok := autocompleteHandlers[parentCommandData.Name]; ok {
				ctx = utils.SetLoggerToContext(ctx, logger.WithFields(log.Fields{
					"command": parentCommandData.Options[0].Name,
				}))
				err = handler(ctx, s, i)
			}
		case discordgo.InteractionMessageComponent:
			commandSplit := strings.Split(i.MessageComponentData().CustomID, ":")
			command := commandSplit[0]
			subcommand := commandSplit[1]

			logger = logger.WithFields(log.Fields{
				"interaction_type": "message command",
				"custom_id":        i.MessageComponentData().CustomID,
				"button_command":   command,
			})

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
			logger = logger.WithFields(log.Fields{
				"interaction_type": "modal submit",
			})
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
					_, _ = b.ChannelMessageSendComplex(settings.GetString("DISCORD.ERROR_CHANNEL_ID"), &discordgo.MessageSend{
						Content: "Ran into an error",
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "Error",
								Description: err.Error(),
								Fields: []*discordgo.MessageEmbedField{
									{Name: "Command", Value: i.Interaction.Type.String()},
									{Name: "Who ran the command", Value: i.Member.User.Mention()},
									{Name: "When", Value: time.Now().Format("2006-01-02 15:04:05 -0700 MST")},
									{Name: "Error", Value: err.Error()},
								},
							},
						},
					})
				}
			}

			switch i.Interaction.Type {
			case discordgo.InteractionApplicationCommand:
				logger.WithFields(log.Fields{
					"command_data": i.ApplicationCommandData(),
				}).WithError(err).Error("running command")
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}); err != nil {
					logger.WithError(err).Warn("creating interaction response")
					if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					}); err != nil {
						logger.WithError(err).Error("creating followup message")
					}
				}
			case discordgo.InteractionMessageComponent:
				logger.WithFields(log.Fields{
					"component_data": i.MessageComponentData(),
				}).WithError(err).Error("running command")
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}); err != nil {
					logger.WithError(err).Warn("creating component interaction response")
					if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Content: msg,
						Flags:   discordgo.MessageFlagsEphemeral,
					}); err != nil {
						logger.WithError(err).Error("creating component followup message")
					}
				}
			case discordgo.InteractionModalSubmit:
				logger.WithFields(log.Fields{
					"modal_data": i.ModalSubmitData(),
				}).WithError(err).Error("running command")
				if _, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				}); err != nil {
					logger.WithError(err).Error("creating component followup message")
				}
			default:
				logger.WithField("interaction_type", i.Interaction.Type).Error("unknown interaction type")
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
	log.Debug("creating profile command")
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

	log.Debug("creating help command")
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "View help",
	}); err != nil {
		return errors.Wrap(err, "creating help command")
	}

	// rank up
	log.Debug("creating rankup command")
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "rankups",
		Description: "Rank up your RSI profile",
	}); err != nil {
		return errors.Wrap(err, "creating rankup command")
	}

	// attendance
	if settings.GetBool("FEATURES.ATTENDANCE.ENABLE") {
		log.Debug("using attendance feature")

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
		log.Debug("using merit feature")
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
		log.Debug("using tokens feature")
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
		log.Debug("using raffles feature")
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
		log.Debug("using giveaways feature")
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

	return b.Open()
}

func (b *Bot) Close() error {
	log.Info("stopping bot")

	// clear commands
	cmds, err := b.ApplicationCommands(b.ClientId, b.GuildId)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		log.WithField("command", cmd.Name).Debug("deleting command")
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, cmd.ID); err != nil {
			return err
		}
	}

	return b.Session.Close()
}

func allowed(discordMember *discordgo.Member, feature string) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
