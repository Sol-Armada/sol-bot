package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

type Bot struct {
	GuildId string

	s *discordgo.Session
}

var bot *Bot
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"attendance": handlers.AttendanceCommandHandler,
	"event":      handlers.EventCommandHandler,
}
var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"event": handlers.EventInteractionHandler,
}

func New() (*Bot, error) {
	if bot == nil {
		log.Info("creating new discord bot")
		b, err := discordgo.New(fmt.Sprintf("Bot %s", config.GetString("DISCORD.BOT_TOKEN")))
		if err != nil {
			return nil, err
		}

		bot = &Bot{
			config.GetString("DISCORD.GUILD_ID"),
			b,
		}

		if _, err := bot.s.Guild(bot.GuildId); err != nil {
			return nil, err
		}

		bot.s.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates

		// command and interaction hanlders
		bot.s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			case discordgo.InteractionMessageComponent:
				id := strings.Split(i.MessageComponentData().CustomID, ":")[0]

				if h, ok := interactionHandlers[id]; ok {
					h(s, i)
				}
			}
		})

		// watch for voice connections and manage them accordingly
		bot.s.AddHandler(handlers.OnVoiceJoin)
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
		return errors.Wrap(err, "failed to start the bot")
	}

	// register commands
	if _, err := b.s.ApplicationCommandCreate(config.GetString("DISCORD.CLIENT_ID"), config.GetString("DISCORD.GUILD_ID"), &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Get your Event Attendence count",
	}); err != nil {
		return errors.Wrap(err, "failed creating attendance command")
	}

	if _, err := b.s.ApplicationCommandCreate(config.GetString("DISCORD.CLIENT_ID"), config.GetString("DISCORD.GUILD_ID"), &discordgo.ApplicationCommand{
		Name:        "event",
		Description: "Event Actions",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "attendance",
				Description: "Take attendance of an Event going on now",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}); err != nil {
		return errors.Wrap(err, "failed creating event command")
	}

	// channels, err := b.GuildChannels(config.GetString("DISCORD.GUILD_ID"))
	// if err != nil {
	// 	log.WithError(err).Error("getting active threads")
	// 	return
	// }

	// for _, channel := range channels {
	// 	if err := b.State.ChannelAdd(channel); err != nil {
	// 		log.WithError(err).Error("adding channel to state")
	// 		return
	// 	}
	// }
	return nil
}

func (b *Bot) Close() error {
	return b.s.Close()
}

func (b *Bot) Monitor() {
	log.Debug("monitoring discord for users")
	for {
		if stores.Storage == nil {
			log.Debug("storage not setup, waiting a bit")
			time.Sleep(10 * time.Second)
			continue
		}

		// rate limit protection
		rateBucket := b.s.Ratelimiter.GetBucket("guild_member_check")
		if rateBucket.Remaining == 0 {
			log.Warn("hit a rate limit. relaxing until it goes away")
			time.Sleep(b.s.Ratelimiter.GetWaitTime(rateBucket, 0))
			continue
		}

		// actually do the members update
		if err := b.UpdateMembers(); err != nil {
			log.WithError(err).Error("getting and storing members")
		}

		time.Sleep(15 * time.Minute)
	}
}

func (b *Bot) UpdateMembers() error {
	log.Debug("checking users")

	m, err := b.GetMembers()
	if err != nil {
		return errors.Wrap(err, "bot getting members")
	}

	storedUsers := []users.User{}
	cur, err := stores.Storage.GetUsers()
	if err != nil {
		return errors.Wrap(err, "getting users for updating")
	}
	if err := cur.All(context.Background(), &storedUsers); err != nil {
		return errors.Wrap(err, "getting users from collection for update")
	}

	for _, member := range m {
		time.Sleep(250 * time.Millisecond)

		u := users.New(member)

		// get the user's primary org, if the nickname is an RSI handle
		po, rank, err := rsi.GetOrgInfo(u.GetTrueNick())
		if err != nil {
			if !errors.Is(err, rsi.UserNotFound) {
				return errors.Wrap(err, "getting rsi based rank")
			}

			u.RSIMember = false
		}
		if member.User.Bot {
			rank = ranks.Bot
		}
		u.Rank = rank
		u.PrimaryOrg = po

		for _, su := range storedUsers {
			if member.User.ID == su.ID {
				u.Events = su.Events
				u.Notes = su.Notes
				u.Ally = su.Ally
				break
			}
		}

		if err := u.Save(); err != nil {
			return errors.Wrap(err, "saving new user")
		}
	}

	return nil
}

func (b *Bot) UpdateMember() error {
	return nil
}

func (b *Bot) GetMembers() ([]*discordgo.Member, error) {
	members, err := b.s.GuildMembers(b.GuildId, "", 1000)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild members")
	}

	return members, nil
}

func (b *Bot) GetMember(id string) (*discordgo.Member, error) {
	member, err := b.s.GuildMember(b.GuildId, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild member")
	}

	return member, nil
}
