package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/bot/attendancehandler"
	"github.com/sol-armada/sol-bot/bot/blueprinthandler"
	"github.com/sol-armada/sol-bot/bot/giveawayhandler"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/bot/jobs"
	"github.com/sol-armada/sol-bot/bot/loothandler"
	"github.com/sol-armada/sol-bot/bot/profilehandler"
	"github.com/sol-armada/sol-bot/bot/rafflehandler"
	"github.com/sol-armada/sol-bot/bot/rankupshandler"
	"github.com/sol-armada/sol-bot/bot/settingshandler"
	"github.com/sol-armada/sol-bot/bot/tokenshandler"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/database"
	"github.com/sol-armada/sol-bot/database/dbnotify"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

type Bot struct {
	GuildId  string
	ClientId string

	botVersion string
	logger     *slog.Logger
	ctx        context.Context

	schedular  *gocron.Scheduler
	dbListener *dbnotify.Listener

	*discordgo.Session
}

type Handler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error

var bot *Bot

var commands = map[string]command.ApplicationCommand{
	"attendance": attendancehandler.New(),
	"profile":    profilehandler.New(),
	"raffle":     rafflehandler.New(),
	"giveaway":   giveawayhandler.New(),
	"tokens":     tokenshandler.New(),
	"rankups":    rankupshandler.New(),
	"blueprint":  blueprinthandler.New(),
	"settings":   settingshandler.New(),
	"loot":       loothandler.New(),
	// "merit":      merithandler.New(),
	// "demerit":    demerithandler.New(),
	// "wikelo":     wikelohandler.New(),
	// "yourelate":  yourelatehandler.New(),
}

var aliases = map[string]command.ApplicationCommand{
	"bp": commands["blueprint"],
}

var onboardingButtonHanlders = map[string]Handler{
	"choice":   onboardingButtonHandler,
	"tryagain": onboardingTryAgainHandler,
}

var onboardingModalHandlers = map[string]Handler{
	"onboard":   onboardingModalHandler,
	"rsihandle": onboardingTryAgainModalHandler,
}

var validateButtonHandlers = map[string]Handler{
	"start": startValidateButtonHandler,
	"check": checkValidateButtonHandler,
}

func New(version string, dbConfig database.Config) (*Bot, error) {
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

	dbListener, err := dbnotify.NewListenerFromPostgresConfig(dbConfig, []dbnotify.Channel{dbnotify.ChannelAttendance})
	if err != nil {
		return nil, err
	}

	bot = &Bot{
		GuildId:    settings.GetString("DISCORD.GUILD_ID"),
		ClientId:   settings.GetString("DISCORD.CLIENT_ID"),
		botVersion: version,
		dbListener: dbListener,
		logger:     slog.Default(),
		ctx:        context.Background(),
		Session:    b,
	}

	return bot, nil
}

func (b *Bot) Setup() error {
	pg := database.Get()
	if pg == nil || pg.Queries == nil {
		return errors.New("database client not initialized")
	}

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

	for _, cmd := range commands {
		cmdData, err := cmd.Setup()
		if err != nil {
			return errors.Wrap(err, "setting up command")
		}

		b.logger.Debug("setting up command", "command", cmd.Name())

		if cmdData == nil {
			b.logger.Warn("command setup returned nil", "command", cmd.Name())
			continue
		}

		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, cmdData); err != nil {
			return errors.Wrap(err, "creating command")
		}

		aliases, err := cmd.SetupAliases()
		if err != nil {
			return errors.Wrap(err, "setting up command aliases")
		}

		if len(aliases) > 0 {
			continue
		}

		b.logger.Debug("setting up command aliases", "command", cmd.Name(), "aliases", len(aliases))

		for _, alias := range aliases {
			b.logger.Debug("creating command alias", "command", cmd.Name(), "alias", alias.Name)

			if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, alias); err != nil {
				return errors.Wrap(err, "creating command alias")
			}
		}
	}

	b.logger.Debug("adding interaction handler")

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		var id string
		switch {
		case i.Member != nil:
			id = i.Member.User.ID
		case i.User != nil:
			id = i.User.ID
		default:
			b.logger.Error("interaction with no user or member", "interaction_type", i.Type)
			return
		}
		member, err := members.Get(id)
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
			"user", id,
			"interaction_type", i.Type,
		)

		logger.Debug("received interaction", "interaction_type", i.Type)

		ctx = utils.SetLoggerToContext(ctx, logger)

		var commandName string
		switch i.Type {
		case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
			commandName = i.ApplicationCommandData().Name
		case discordgo.InteractionMessageComponent:
			commandSplit := strings.Split(i.MessageComponentData().CustomID, ":")
			commandName = commandSplit[0]
		case discordgo.InteractionModalSubmit:
			commandSplit := strings.Split(i.ModalSubmitData().CustomID, ":")
			commandName = commandSplit[0]
		default:
			logger.Error("unknown interaction type", "interaction_type", i.Type)
			return
		}

		if cmd, ok := commands[commandName]; ok {
			cmdToStore := command.Command{
				Name: commandName,
				User: id,
				When: time.Now(),
				Type: i.Type,
			}

			if i.Type == discordgo.InteractionApplicationCommand {
				cmdToStore.Options = i.ApplicationCommandData().Options
			}

			if i.Type == discordgo.InteractionMessageComponent {
				cmdToStore.ButtonId = i.MessageComponentData().CustomID
			}

			if err := command.RunCommand(ctx, cmd, s, i); err != nil {
				logger.Error("running command", "command", commandName, "error", err)
				cmdToStore.Error = err.Error()

				if _, err := b.ChannelMessageSendComplex(settings.GetString("DISCORD.ERROR_CHANNEL_ID"), &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Error",
							Description: err.Error(),
							Fields: []*discordgo.MessageEmbedField{
								{Name: "Who ran the command", Value: i.Member.User.Mention(), Inline: false},
								{Name: "Command", Value: commandName, Inline: true},
							},
						},
					},
				}); err != nil {
					logger.Error("sending error message", "error", err)
					return
				}

				if i.Type != discordgo.InteractionMessageComponent {
					msg, err := s.InteractionResponse(i.Interaction)
					if err != nil {
						logger.Error("getting interaction response", "error", err)
						return
					}

					if _, err := s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
						Content: new("I ran into an error! You didn't do anything wrong. I have notified the right people and hopfully this gets fixed."),
					}); err != nil {
						logger.Error("editing followup message", "error", err)
					}
				}
			}
			if i.Type != discordgo.InteractionApplicationCommandAutocomplete {
				optionsJSON := "[]"
				if len(cmdToStore.Options) > 0 {
					if raw, err := json.Marshal(cmdToStore.Options); err == nil {
						optionsJSON = string(raw)
					}
				}

				if err := pg.Queries.InsertCommandLog(ctx, dbgen.InsertCommandLogParams{
					Name:            cmdToStore.Name,
					OccurredAt:      toPgTs(cmdToStore.When),
					UserID:          cmdToStore.User,
					InteractionType: int32(cmdToStore.Type),
					ButtonID:        cmdToStore.ButtonId,
					ErrorText:       cmdToStore.Error,
					OptionsJson:     optionsJSON,
				}); err != nil {
					logger.Error("creating command record", "command", commandName, "error", err)
				}
			}

			return
		}

		if cmd, ok := aliases[commandName]; ok {
			if err := command.RunCommand(ctx, cmd, s, i); err != nil {
				logger.Error("running command alias", "alias", commandName, "error", err)
				if _, err := b.ChannelMessageSendComplex(settings.GetString("DISCORD.ERROR_CHANNEL_ID"), &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Error",
							Description: err.Error(),
							Fields: []*discordgo.MessageEmbedField{
								{Name: "Who ran the command", Value: i.Member.User.Mention(), Inline: false},
								{Name: "Command Alias", Value: commandName, Inline: true},
							},
						},
					},
				}); err != nil {
					logger.Error("sending error message", "error", err)
					return
				}

				if i.Type != discordgo.InteractionMessageComponent {
					msg, err := s.InteractionResponse(i.Interaction)
					if err != nil {
						logger.Error("getting interaction response", "error", err)
						return
					}

					response := "I ran into an error! You didn't do anything wrong. I have notified the right people and hopfully this gets fixed."
					switch {
					case errors.Is(err, rsi.ErrForbidden):
						response = "RSI is currently not allowing me to access user profiles. Give it some time and try again."
					}
					if _, err := s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
						Content: &response,
					}); err != nil {
						logger.Error("editing followup message", "error", err)
					}
				}
			}
			return
		}

		logger.Warn("no command found for interaction", "interaction_type", i.Type, "command_name", commandName)

		switch i.Type {
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
			default:
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
			case rsi.ErrForbidden:
				msg = "RSI is currently not allowing me to access user profiles. Give it some time and try again"
			default:
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
	// watch for on join and leave
	b.AddHandler(onJoinHandler)
	b.AddHandler(onLeaveHandler)
	b.AddHandler(OnNameChangeHandler)

	if err := setupOnboarding(); err != nil {
		return errors.Wrap(err, "setting up onboarding")
	}

	// giveaways
	if err := giveaway.Load(b.Session); err != nil {
		return errors.Wrap(err, "loading giveaways")
	}

	// activity tracking
	b.AddHandler(onVoiceUpdate)

	b.logger.Debug("opening Discord connection")

	if err := b.Open(); err != nil {
		b.logger.Error("failed to open Discord connection", "error", err)
		return errors.Wrap(err, "opening Discord connection")
	}

	b.logger.Debug("Discord connection opened successfully")

	go b.startJobs()

	go b.StartAttendanceWatch()

	return nil
}

type statusJobMonitor struct {
	id string
}

func newStatusJobMonitor(jobName string) statusJobMonitor {
	return statusJobMonitor{id: jobName}
}

func (m statusJobMonitor) Update(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}

	upsertStatusMessage(m.id, message)
}

func (m statusJobMonitor) Done() {
	removeStatusMessage(m.id)
}

func (b *Bot) startJobs() {
	s, err := gocron.NewScheduler(
		gocron.WithLogger(b.logger),
	)
	if err != nil {
		b.logger.Error("failed to create scheduler", "error", err)
		return
	}
	b.schedular = &s

	for _, job := range jobs.GetJobs() {
		if _, err = s.NewJob(
			gocron.CronJob(job.Cron, false),
			gocron.NewTask(
				func(ctx context.Context) error {
					monitor := newStatusJobMonitor(job.Name)
					monitor.Update(fmt.Sprintf("Running job %s", job.Name))
					defer monitor.Done()

					return job.Run(ctx, b.Session, monitor)
				},
			),
			gocron.WithName(job.Name),
		); err != nil {
			b.logger.Error("failed to create job", "job_name", job.Name, "error", err)
			return
		}

		b.logger.Info("scheduled job", "job_name", job.Name, "cron", job.Cron)
	}

	s.Start()

	b.logger.Info("started jobs")
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
		if slices.Contains(slices.Collect(maps.Keys(commands)), cmd.Name) || slices.Contains(slices.Collect(maps.Keys(aliases)), cmd.Name) {
			continue
		}

		b.logger.Debug("deleting command", "command", cmd.Name)
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, cmd.ID); err != nil {
			return err
		}
	}

	return b.Session.Close()
}

func (b *Bot) UpdateCustomStatus(status string) error {
	if status == "" {
		status = fmt.Sprintf("ready to serve - (%s)", b.botVersion)
	}
	return b.Session.UpdateCustomStatus(status)
}

func toPgTs(v time.Time) pgtype.Timestamptz {
	if v.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: v.UTC(), Valid: true}
}
