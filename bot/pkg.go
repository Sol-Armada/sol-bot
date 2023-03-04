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
	"golang.org/x/exp/slices"
)

type Bot struct {
	GuildId  string
	ClientId string

	s *discordgo.Session
}

var bot *Bot
var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"choice": handlers.ChoiceButtonHandler,
	// "guest_friend": handlers.GuestFriendHandler,
	"start_over": handlers.StartOverHandler,
	"event":      handlers.EventInteractionHandler,
}
var modalHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"rsi_handle": handlers.RSIModalHandler,
	// "friend_of":  handlers.GuestFriendOfModalHandler,
	// "ally_org":   handlers.AllyOrgModalHandler,
}
var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"attendance": handlers.AttendanceCommandHandler,
	"event":      handlers.EventCommandHandler,
	"onboarding": handlers.OnboardingCommandHandler,
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
			case discordgo.InteractionMessageComponent:
				id := i.MessageComponentData().CustomID
				if strings.HasPrefix(id, "choice") {
					interactionHandlers["choice"](s, i)
				}
				if strings.HasPrefix(id, "start_over") {
					interactionHandlers["start_over"](s, i)
				}
				if strings.HasPrefix(id, "guest_friend") {
					interactionHandlers["guest_friend"](s, i)
				}
			case discordgo.InteractionModalSubmit:
				id := i.ModalSubmitData().CustomID
				if strings.HasPrefix(id, "rsi_handle") {
					modalHandlers["rsi_handle"](s, i)
				}
				if strings.HasPrefix(id, "friend_of") {
					modalHandlers["friend_of"](s, i)
				}
				if strings.HasPrefix(id, "ally_org") {
					modalHandlers["ally_org"](s, i)
				}
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			}
		})

		// watch for voice connections and manage them accordingly
		bot.s.AddHandler(handlers.OnVoiceJoin)
		// watch for server join
		bot.s.AddHandler(handlers.JoinServerHandler)
		// watch for server leave
		bot.s.AddHandler(handlers.LeaveServerHandler)
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
	if _, err := b.s.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Get your Event Attendence count",
	}); err != nil {
		return errors.Wrap(err, "failed creating attendance command")
	}

	if _, err := b.s.ApplicationCommandCreate(b.ClientId, b.GuildId, &discordgo.ApplicationCommand{
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

func (b *Bot) Monitor(stop <-chan bool, done chan bool) {
	log.Info("monitoring discord for users")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastChecked := time.Now().Add(-30 * time.Minute)
	for {
		select {
		case <-stop:
			log.Info("stopping monitor")
			goto DONE
		case <-ticker.C:
			if time.Now().After(lastChecked.Add(30 * time.Minute)) {
				log.Info("scanning users")
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

				lastChecked = time.Now()
			}

			continue
		}
	DONE:
		break
	}
	done <- true
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
				break
			}
		}

		// get the user's primary org, if the nickname is an RSI handle
		trueNick := u.GetTrueNick()
		po, ao, rank, err := rsi.GetOrgInfo(trueNick)
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

		for _, affiliatedOrg := range ao {
			if slices.Contains(config.GetStringSlice("enemies"), affiliatedOrg) {
				u.BadAffiliation = true
				u.Rank = ranks.Guest
				break
			}
		}

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
