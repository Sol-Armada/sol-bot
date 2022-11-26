package bot

import (
	"context"
	"fmt"
	"regexp"
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
var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice":       ChoiceButtonHandler,
	"guest_friend": GuestFriendHandler,
	"start_over":   StartOverHandler,
	"event": handlers.EventInteractionHandler,
}
var modalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"rsi_handle": RSIModalHandler,
	"friend_of":  GuestFriendOfModalHandler,
	"ally_org":   AllyOrgModalHandler,
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"attendance": handlers.AttendanceCommandHandler,
	"event":      handlers.EventCommandHandler,
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

		bot.AddHandler(JoinServerHandler)
		bot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
			case discordgo.InteractionMessageComponent:
				// if h, ok := interactionHandlers[i.MessageComponentData().CustomID]; ok {
				// 	h(s, i)
				// }

				id := strings.Split(i.MessageComponentData().CustomID, ":")[0]
				if h, ok := interactionHandlers[id]; ok {
					h(s, i)
				}

				cid := i.MessageComponentData().CustomID
				if strings.HasPrefix(cid, "choice") {
					interactionHandlers["choice"](s, i)
				}
				if strings.HasPrefix(cid, "start_over") {
					interactionHandlers["start_over"](s, i)
				}
				if strings.HasPrefix(cid, "guest_friend") {
					interactionHandlers["guest_friend"](s, i)
				}
			case discordgo.InteractionModalSubmit:
				cid := i.ModalSubmitData().CustomID
				if strings.HasPrefix(cid, "rsi_handle") {
					modalHandlers["rsi_handle"](s, i)
				}
				if strings.HasPrefix(cid, "friend_of") {
					modalHandlers["friend_of"](s, i)
				}
				if strings.HasPrefix(cid, "ally_org") {
					modalHandlers["ally_org"](s, i)
				}
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			}
		})

		// command and interaction hanlders
		bot.s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.Type {
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

		// get the discord members
		m, err := b.GetMembers()
		if err != nil {
			log.WithError(err).Error("bot getting members")
			return
		}

		// get the stored members
		storedUsers := []*users.User{}
		cur, err := stores.Storage.GetUsers()
		if err != nil {
			log.WithError(err).Error("getting users for updating")
			return
		}
		if err := cur.All(context.Background(), &storedUsers); err != nil {
			log.WithError(err).Error("getting users from collection for update")
			return
		}

		// actually do the members update
		if err := updateMembers(m, storedUsers); err != nil {
			if strings.Contains(err.Error(), "Forbidden") {
				log.Warn("we hit the limit with RSI's website. let's wait and try again...")
				time.Sleep(30 * time.Minute)
				continue
			}

			log.WithError(err).Error("updating members")
			return
		}

		// do some cleaning
		if err := cleanMembers(m, storedUsers); err != nil {
			log.WithError(err).Error("cleaning up the members")
			return
		}

		time.Sleep(30 * time.Minute)
	}
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

func updateMembers(m []*discordgo.Member, storedUsers []*users.User) error {
	log.Debug("checking users")

	for _, member := range m {
		time.Sleep(500 * time.Millisecond)

		u := users.New(member)

		for _, su := range storedUsers {
			if member.User.ID == su.ID {
				u.Events = su.Events
				u.Notes = su.Notes
				u.Ally = su.Ally
				break
			}
		}

		if u.Ally {
			continue
		}

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

		for _, a := range config.GetStringSlice("allies") {
			if strings.EqualFold(u.PrimaryOrg, a) {
				u.Rank = ranks.Ally
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
	members, err := bot.GuildMembers(b.GuildId, "", 1000)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild members")	
	}

	return nil
}

func cleanMembers(m []*discordgo.Member, storedUsers []*users.User) error {
	for _, user := range storedUsers {
		for _, member := range m {
			if user.ID == member.User.ID {
				goto CONTINUE
			}
		}

		log.WithField("user", user).Info("deleting user")
		if err := stores.Storage.DeleteUser(user.ID); err != nil {
			return errors.Wrap(err, "cleaning members")
		}
	CONTINUE:
		continue
	}

	return nil
}
