package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/settings"
	"github.com/sol-armada/admin/utils"
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
}

var autocompleteHandlers = map[string]Handler{
	"takeattendance":   takeAttendanceAutocompleteHandler,
	"removeattendance": removeAttendanceAutocompleteHandler,
}

var attendanceButtonHandlers = map[string]Handler{
	"record":  recordAttendanceButtonHandler,
	"recheck": recheckIssuesButtonHandler,
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

	b.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates + discordgo.IntentsGuildMessageReactions

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
	log.Debug("bot is ready")
}

func (b *Bot) Setup() error {
	// setup state when bot is ready
	b.AddHandler(ready)

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		member, err := members.Get(i.Member.User.ID)
		if err != nil {
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
			}
		}

		if err != nil {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: "It looks like I ran into an error. I have logged it and someone will look into it. Please try again in a few minutes.",
				},
			})
		}
	})

	// watch for on join
	b.AddHandler(onJoinHandler)

	// clear commands
	cmds, err := b.ApplicationCommands(b.ClientId, b.GuildId)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		log.WithField("command", cmd.Name).Info("deleting command")
		// if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, cmd.ID); err != nil {
		// 	return err
		// }
	}

	// register commands

	// misc commands
	if err := b.DeleteCommand("attendance"); err != nil {
		log.WithError(err).Error("unable to delete attendance command")
	}
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "[DEPRICATED] use /profile instead",
	}); err != nil {
		return errors.Wrap(err, "creating attendance command")
	}
	if err := b.DeleteCommand("profile"); err != nil {
		log.WithError(err).Error("unable to delete profile command")
	}
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "View your profile",
	}); err != nil {
		return errors.Wrap(err, "creating profile command")
	}

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

	// bank
	if err := b.DeleteCommand("bank"); err != nil {
		log.WithError(err).Error("unable to delete bank command")
	}
	if settings.GetBoolWithDefault("FEATURES.BANK", false) {
		log.Info("using bank feature")
		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "bank",
			Description: "Manage the bank",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "balance",
					Description: "How much is in the bank",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "add",
					Description: "Add aUEC to the bank",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "from",
							Description:  "who the money came from",
							Type:         discordgo.ApplicationCommandOptionMentionable,
							Required:     true,
							Autocomplete: true,
						},
						{
							Name:        "amount",
							Description: "how much",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
						{
							Name:        "notes",
							Description: "extra information",
							Type:        discordgo.ApplicationCommandOptionString,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove aURC from the bank",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "to",
							Description:  "who the money is going to",
							Type:         discordgo.ApplicationCommandOptionMentionable,
							Required:     true,
							Autocomplete: true,
						},
						{
							Name:        "amount",
							Description: "how much",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
						{
							Name:        "notes",
							Description: "extra information",
							Type:        discordgo.ApplicationCommandOptionString,
						},
					},
				},
				{
					Name:        "spend",
					Description: "Spend aUEC",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "for",
							Description: "what you spending aUEC on",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "amount",
							Description: "how much",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
						},
						{
							Name:        "notes",
							Description: "extra information",
							Type:        discordgo.ApplicationCommandOptionString,
						},
					},
				},
			},
		}); err != nil {
			return errors.Wrap(err, "failed creating bank command")
		}
	}

	if err := b.DeleteCommand("merit"); err != nil {
		log.WithError(err).Error("unable to delete merit command")
	}
	if err := b.DeleteCommand("demerit"); err != nil {
		log.WithError(err).Error("unable to delete demerit command")
	}
	if settings.GetBoolWithDefault("FEATURES.MERIT.ENABLED", false) {
		log.Info("using merit feature")
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

	return b.Open()
}

func (b *Bot) Close() error {
	log.Info("stopping bot")
	if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, "onboarding"); err != nil {
		return errors.Wrap(err, "failed deleting oboarding command")
	}
	return b.Close()
}

func (b *Bot) SendMessage(channelId string, message string) (*discordgo.Message, error) {
	return b.ChannelMessageSend(channelId, message)
}

func (b *Bot) SendComplexMessage(channelId string, message *discordgo.MessageSend) (*discordgo.Message, error) {
	return b.ChannelMessageSendComplex(channelId, message)
}

func (b *Bot) GetEmojis() ([]*discordgo.Emoji, error) {
	return b.GuildEmojis(b.GuildId)
}

func (b *Bot) DeleteEventMessage(id string) error {
	return b.ChannelMessageDelete(settings.GetString("DISCORD.CHANNELS.EVENTS"), id)
}

func (b *Bot) DeleteCommand(name string) error {
	cmds, err := b.ApplicationCommands(b.ClientId, b.GuildId)
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		if cmd.Name == name {
			if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, cmd.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
