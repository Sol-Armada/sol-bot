package bot

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/bot/handlers/event"
	"github.com/sol-armada/admin/bot/handlers/onboarding"
	"github.com/sol-armada/admin/config"
)

type Bot struct {
	GuildId  string
	ClientId string

	s *discordgo.Session
}

var bot *Bot

// command handlers
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"event":      event.EventCommandHandler,
	"onboarding": onboarding.OnboardingCommandHandler,
	"attendance": event.AttendanceCommandHandler,
}

// event hanlders
var eventInteractionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"event": event.EventInteractionHandler,
}

// onboarding handlers
var onboardingInteractionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice":     onboarding.ChoiceButtonHandler,
	"start_over": onboarding.StartOverHandler,
}
var onboardingModalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"rsi_handle": onboarding.RSIModalHandler,
}

func New() (*Bot, error) {
	if bot == nil {
		log.Info("creating discord bot")
		b, err := discordgo.New(fmt.Sprintf("Bot %s", config.GetString("DISCORD.BOT_TOKEN")))
		if err != nil {
			return nil, err
		}

		bot = &Bot{
			config.GetString("DISCORD.GUILD_ID"),
			config.GetString("DISCORD.CLIENT_ID"),
			b,
		}

		if _, err := bot.s.Guild(bot.GuildId); err != nil {
			return nil, err
		}

		bot.s.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates

		bot.s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			case discordgo.InteractionMessageComponent:
				id := strings.Split(i.MessageComponentData().CustomID, ":")

				switch id[0] {
				case "event":
					eventInteractionHandlers[id[1]](s, i)
				case "onboarding":
					onboardingInteractionHandlers[id[1]](s, i)
				}
			case discordgo.InteractionModalSubmit:
				id := strings.Split(i.ModalSubmitData().CustomID, ":")

				switch id[0] {
				case "onboarding":
					onboardingModalHandlers[id[1]](s, i)
				}
			}
		})

		// watch for voice connections and manage them accordingly
		bot.s.AddHandler(event.OnVoiceJoin)
		// watch for server join
		bot.s.AddHandler(onboarding.JoinServerHandler)
		// watch for server leave
		bot.s.AddHandler(onboarding.LeaveServerHandler)
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

func (b *Bot) Open() error {
	if err := b.s.Open(); err != nil {
		return errors.Wrap(err, "starting the bot")
	}

	// register commands

	if _, err := b.s.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Get your attendance count",
	}); err != nil {
		return errors.Wrap(err, "creating attendance command")
	}

	// event
	if config.GetBoolWithDefault("FEATURES.EVENT", false) {
		log.Debug("using event feature")

		if _, err := b.s.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "event",
			Description: "Event Actions",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new event",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "attendance",
					Description: "Take attendance of an Event going on now",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		}); err != nil {
			return errors.Wrap(err, "creating event command")
		}
	} else {
		if err := b.s.ApplicationCommandDelete(b.ClientId, b.GuildId, "event"); err != nil {
			log.WithError(err).Warn("deleting event command")
		}
	}

	// onboarding
	if config.GetBoolWithDefault("FEATURES.ONBOARDING", false) {
		log.Debug("using onboarding feature")
		if _, err := b.s.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "onboarding",
			Description: "Start onboarding process for someone",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "single-autcomplete",
					Description:  "the member to onboard",
					Type:         discordgo.ApplicationCommandOptionMentionable,
					Required:     true,
					Autocomplete: true,
				},
			},
		}); err != nil {
			return errors.Wrap(err, "failed creating oboarding command")
		}
	} else {
		if err := b.s.ApplicationCommandDelete(b.ClientId, b.GuildId, "onboarding"); err != nil {
			log.WithError(err).Warn("deleting onboarding command")
		}
	}

	return nil
}

func (b *Bot) Close() error {
	log.Info("stopping bot")
	if err := b.s.ApplicationCommandDelete(b.ClientId, b.GuildId, "onboarding"); err != nil {
		return errors.Wrap(err, "failed deleting oboarding command")
	}
	return b.s.Close()
}
