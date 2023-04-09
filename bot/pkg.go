package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/kyokomi/emoji/v2"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/bot/handlers/bank"
	"github.com/sol-armada/admin/bot/handlers/event"
	"github.com/sol-armada/admin/bot/handlers/onboarding"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
)

type Bot struct {
	GuildId  string
	ClientId string

	*discordgo.Session
}

var bot *Bot

var NextEvent *events.Event

// command handlers
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"onboarding": onboarding.OnboardingCommandHandler,
	"bank":       bank.BankCommandHandler,
	"attendance": handlers.AttendanceCommandHandler,
}

// event hanlders
var eventInteractionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"position": event.PositionHandler,
}

// onboarding handlers
var onboardingInteractionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice":               onboarding.ChoiceButtonHandler,
	"try_rsi_handle_again": onboarding.TryAgainHandler,
}
var onboardingModalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"rsi_handle": onboarding.RSIModalHandler,
}

func New() (*Bot, error) {
	log.Info("creating discord bot")
	b, err := discordgo.New(fmt.Sprintf("Bot %s", config.GetString("DISCORD.BOT_TOKEN")))
	if err != nil {
		return nil, err
	}

	if _, err := b.Guild(config.GetString("DISCORD.GUILD_ID")); err != nil {
		return nil, err
	}

	b.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates

	bot = &Bot{
		config.GetString("DISCORD.GUILD_ID"),
		config.GetString("DISCORD.CLIENT_ID"),
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
	defer func() {
		if err := b.Open(); err != nil {
			log.WithError(err).Error("starting the bot")
		}
	}()

	// setup state when bot is ready
	b.AddHandler(ready)

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// register commands

	// misc commands
	if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Get your attendance count",
	}); err != nil {
		return errors.Wrap(err, "creating attendance command")
	}

	// onboarding
	if config.GetBoolWithDefault("FEATURES.ONBOARDING", false) {
		log.Info("using onboarding feature")

		if err := b.SetupOnboarding(); err != nil {
			return errors.Wrap(err, "failed to setup onboarding")
		}

		if _, err := b.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
			Name:        "onboarding",
			Description: "Start onboarding process for someone",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "member",
					Description:  "the member to onboard",
					Type:         discordgo.ApplicationCommandOptionMentionable,
					Required:     true,
					Autocomplete: true,
				},
			},
		}); err != nil {
			return errors.Wrap(err, "failed creating oboarding command")
		}
		// watch for server join
		b.AddHandler(onboarding.JoinServerHandler)
		// watch for server leave
		b.AddHandler(onboarding.LeaveServerHandler)
	} else {
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, "onboarding"); err != nil {
			log.WithError(err).Warn("deleting onboarding command")
		}
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, "join"); err != nil {
			log.WithError(err).Warn("deleting join command")
		}
	}

	// bank
	if config.GetBoolWithDefault("FEATURES.BANK", false) {
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

		// watch for server join
		bot.AddHandler(onboarding.JoinServerHandler)
		// watch for server leave
		bot.AddHandler(onboarding.LeaveServerHandler)
	} else {
		if err := b.ApplicationCommandDelete(b.ClientId, b.GuildId, "bank"); err != nil {
			log.WithError(err).Warn("deleting bank command")
		}
	}

	// events
	if config.GetBoolWithDefault("FEATURES.EVENTS", false) {
		log.Info("using events feature")
		b.AddHandler(event.EventReactionAdd)

		// watch the events
		go b.EventWatcher()
	}

	return nil
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

func (b *Bot) StartEvent(e *events.Event) error {
	logger := log.WithField("event start", e)
	logger.Info("starting event")

	eventChannelId := config.GetString("DISCORD.CHANNELS.EVENTS")

	message, err := b.ChannelMessage(eventChannelId, e.MessageId)
	if err != nil {
		return errors.Wrap(err, "getting original event message")
	}

	thread, err := b.MessageThreadStartComplex(message.ChannelID, message.ID, &discordgo.ThreadStart{
		Name: "Event Started!",
		Type: discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		return errors.Wrap(err, "starting event thread")
	}

	if e.Status != events.Live {
		e.Status = events.Live
		if err := e.Save(); err != nil {
			return errors.Wrap(err, "saving live event")
		}

		// alert the event is live
		if _, err := b.ChannelMessageSend(thread.ID, "Event Starting"); err != nil {
			return errors.Wrap(err, "sending event starting message")
		}
	}

	timer := time.NewTimer(time.Until(e.End))
	<-timer.C

	// stop the event

	// alert the event is over
	if _, err := b.ChannelMessageSend(thread.ID, "Event Over"); err != nil {
		return errors.Wrap(err, "sending event over message")
	}

	e.Status = events.Finished
	if err := e.Save(); err != nil {
		return errors.Wrap(err, "saving finished event")
	}

	message.Embeds[0].Fields[0].Value = fmt.Sprintf("<t:%d> - <t:%d:t>", e.Start.Unix(), e.End.Unix())

	if _, err := b.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         message.ID,
		Channel:    message.ChannelID,
		Components: []discordgo.MessageComponent{},
		Embeds:     message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "updating original event message")
	}

	return nil
}

func (b *Bot) NotifyOfEvent(e *events.Event) error {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Time",
			Value: fmt.Sprintf("<t:%d> - <t:%d:t>\n:timer: <t:%d:R>", e.Start.Unix(), e.End.Unix(), e.Start.Unix()),
		},
	}

	emojis := emoji.CodeMap()

	for _, position := range e.Positions {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s (0/%d)", emojis[":"+position.Emoji+":"], position.Name, position.Max),
			Value:  "-",
			Inline: true,
		})
	}

	if e.Cover == "/logo.png" || e.Cover == "" {
		e.Cover = "https://admin.solarmada.space/logo.png"
	}

	message, err := b.SendComplexMessage(config.GetString("DISCORD.CHANNELS.EVENTS"), &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeArticle,
				Title:       e.Name,
				Description: e.Description,
				Fields:      fields,
				Image: &discordgo.MessageEmbedImage{
					URL: e.Cover,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	for _, p := range e.Positions {
		if err := b.MessageReactionAdd(message.ChannelID, message.ID, emojis[":"+p.Emoji+":"]); err != nil {
			// return err
			fmt.Println(err.Error())
		}
	}

	e.MessageId = message.ID
	if err := e.Save(); err != nil {
		return err
	}

	return nil
}

func (b *Bot) SetupOnboarding() error {
	channels, err := b.GuildChannels(b.GuildId)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.ParentID == config.GetString("DISCORD.CATEGORIES.AIRLOCK") && channel.Name == "onboarding" {
			return nil
		}
	}

	newChannel, err := b.GuildChannelCreateComplex(b.GuildId, discordgo.GuildChannelCreateData{
		Name:     "onboarding",
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: config.GetString("DISCORD.CATEGORIES.AIRLOCK"),
	})
	if err != nil {
		return err
	}

	m := `Welcome to Sol Armada!
	
Select a reason you joined below. We will ask a few questions then assign you a role.`

	if _, err := b.ChannelMessageSendComplex(newChannel.ID, &discordgo.MessageSend{
		Content: m,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "A member recruited me",
						CustomID: "onboarding:choice:recruited",
					},
					discordgo.Button{
						Label:    "Found Sol Armada on RSI",
						CustomID: "onboarding:choice:rsi",
					},
					discordgo.Button{
						Label:    "Some other way",
						CustomID: "onboarding:choice:other",
					},
					discordgo.Button{
						Label:    "Just visiting",
						CustomID: "onboarding:choice:visiting",
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func (b *Bot) ScheduleEvent(e *events.Event) {
	NextEvent = e
	e.Timer = time.NewTimer(time.Until(e.Start))
	<-e.Timer.C

	NextEvent = nil
	if err := b.StartEvent(e); err != nil {
		log.WithError(err).Error("starting event")
	}
}

func (b *Bot) GetEmojis() ([]*discordgo.Emoji, error) {
	return b.GuildEmojis(b.GuildId)
}

func (b *Bot) DeleteEventMessage(id string) error {
	return b.ChannelMessageDelete(config.GetString("DISCORD.CHANNELS.EVENTS"), id)
}
