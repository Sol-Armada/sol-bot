package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

type Bot struct {
	GuildId  string
	ClientId string

	ctx context.Context

	*discordgo.Session
}

type Handler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error

var bot *Bot

// command handlers
var commandHandlers = map[string]Handler{
	"takeattendance":   takeAttendanceCommandHandler,
	"removeattendance": removeAttendanceCommandHandler,
	"profile":          profileCommandHandler,
	"merit":            giveMeritCommandHandler,
	"demerit":          giveDemeritCommandHandler,
	"validate":         validateCommandHandler,
	"rankups":          rankUpsCommandHandler,
}

var autocompleteHandlers = map[string]Handler{
	"takeattendance":   takeAttendanceAutocompleteHandler,
	"removeattendance": removeAttendanceAutocompleteHandler,
}

var onboardingButtonHanlders = map[string]Handler{
	"validate": validateButtonHandler,
	"choice":   onboardingButtonHandler,
	"tryagain": onboardingTryAgainHandler,
}

var onboardingModalHandlers = map[string]Handler{
	"onboard":   onboardingModalHandler,
	"rsihandle": onboardingTryAgainModalHandler,
}

var attendanceButtonHandlers = map[string]Handler{
	"record":       recordAttendanceButtonHandler,
	"recheck":      recheckIssuesButtonHandler,
	"delete":       deleteAttendanceButtonHandler,
	"verifydelete": verifyDeleteButtonModalHandler,
	"canceldelete": cancelDeleteButtonModalHandler,
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
		settings.GetString("DISCORD.GUILD_ID"),
		settings.GetString("DISCORD.CLIENT_ID"),
		context.Background(),
		b,
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

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.State.TrackVoice = true
	log.Info("bot is ready")
}

func (b *Bot) Setup() error {
	// setup state when bot is ready
	b.AddHandler(ready)

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		member, err := members.Get(i.Member.User.ID)
		if err != nil {
			if errors.Is(err, members.MemberNotFound) {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Looks like you are not onboarded yet! Ask an @Officer to help onboard you",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

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
			logger = logger.WithFields(log.Fields{
				"interaction_type": "application command",
				"name":             i.ApplicationCommandData().Name,
			})
			if h, ok := autocompleteHandlers[i.ApplicationCommandData().Name]; ok {
				ctx = utils.SetLoggerToContext(ctx, logger.WithField("interaction_type", "application command autocomplete"))
				err = h(ctx, s, i)
			}
		case discordgo.InteractionMessageComponent:
			logger = logger.WithFields(log.Fields{
				"interaction_type": "message command",
				"custom_id":        i.MessageComponentData().CustomID,
			})
			id := strings.Split(i.MessageComponentData().CustomID, ":")
			ctx = utils.SetLoggerToContext(ctx, logger)
			switch id[0] {
			case "attendance":
				if h, ok := attendanceButtonHandlers[id[1]]; ok {
					err = h(ctx, s, i)
				}
			case "onboarding":
				if h, ok := onboardingButtonHanlders[id[1]]; ok {
					err = h(ctx, s, i)
				}
			}
		case discordgo.InteractionModalSubmit:
			logger = logger.WithFields(log.Fields{
				"interaction_type": "modal submit",
			})
			ctx = utils.SetLoggerToContext(ctx, logger)
			id := strings.Split(i.ModalSubmitData().CustomID, ":")
			switch id[0] {
			case "onboarding":
				if h, ok := onboardingModalHandlers[id[1]]; ok {
					err = h(ctx, s, i)
				}
			}
		}

		if err != nil { // handle any errors returned
			if err == InvalidPermissions {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You don't have permission to do that",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			logger.WithError(err).Error("running command")

			msg := "It looks like I ran into an error. I have logged it and someone will look into it. Ask an @Officer if you need help"
			switch i.Interaction.Type {
			case discordgo.InteractionApplicationCommand:
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

	// validate
	log.Debug("creating validate command")
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "validate",
		Description: "Validate your RSI profile",
	}); err != nil {
		return errors.Wrap(err, "creating validate command")
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

		options := []*discordgo.ApplicationCommandOption{
			{
				Name:         "event",
				Description:  "the event to take attendance for",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		}
		for i := 0; i < 10; i++ {
			o := &discordgo.ApplicationCommandOption{
				Name:         fmt.Sprintf("user-%d", i+1),
				Description:  "the user to take attendance for",
				Type:         discordgo.ApplicationCommandOptionUser,
				Autocomplete: true,
			}
			if i == 0 {
				o.Required = true
			}
			options = append(options, o)
		}
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "takeattendance",
			Description: "take or add to attendance",
			Type:        discordgo.ChatApplicationCommand,
			Options:     options,
		}); err != nil {
			return errors.Wrap(err, "creating takeattendance command")
		}

		options = []*discordgo.ApplicationCommandOption{
			{
				Name:         "event",
				Description:  "the event to remove attendance from",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		}
		for i := 0; i < 10; i++ {
			o := &discordgo.ApplicationCommandOption{
				Name:         fmt.Sprintf("user-%d", i+1),
				Description:  "the user to remove from attedance",
				Type:         discordgo.ApplicationCommandOptionUser,
				Autocomplete: true,
			}
			if i == 0 {
				o.Required = true
			}
			options = append(options, o)
		}
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "removeattendance",
			Description: "remove from attendance",
			Type:        discordgo.ChatApplicationCommand,
			Options:     options,
		}); err != nil {
			return errors.Wrap(err, "creating removeattendance command")
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

	return b.Close()
}
