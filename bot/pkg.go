package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/bot/handlers"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/users"
)

type Bot struct {
	GuildId string

	*discordgo.Session
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

		if _, err := bot.Guild(bot.GuildId); err != nil {
			return nil, err
		}

		bot.Identify.Intents = discordgo.IntentGuildMembers + discordgo.IntentGuildVoiceStates

		bot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		bot.AddHandler(handlers.OnVoiceJoin)
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

func (b *Bot) Monitor() {
	log.Debug("monitoring discord for users")
	for {
		// rate limit protection
		rateBucket := b.Ratelimiter.GetBucket("guild_member_check")
		if rateBucket.Remaining == 0 {
			log.Warn("hit a rate limit. relaxing until it goes away")
			time.Sleep(b.Ratelimiter.GetWaitTime(rateBucket, 0))
			continue
		}

		// actually do the members update
		if err := b.UpdateMembers(); err != nil {
			log.WithError(err).Error("getting and storing members")
		}

		time.Sleep(1 * time.Hour)
	}
}

func (b *Bot) UpdateMembers() error {
	log.Debug("checking users")

	m, err := b.GetMembers()
	if err != nil {
		return errors.Wrap(err, "bot getting members")
	}

	storedUsers, err := users.GetStorage().GetUsers()
	if err != nil {
		return errors.Wrap(err, "getting users for updating")
	}

	for _, member := range m {
		time.Sleep(250 * time.Millisecond)
		nick := member.User.Username
		if member.Nick != "" {
			nick = member.Nick
		}

		u := &users.User{
			Nick:          nick,
			Id:            member.User.ID,
			Username:      member.User.Username,
			Discriminator: member.User.Discriminator,
			Avatar:        member.User.Avatar,
			Ally:          false,
			PrimaryOrg:    "",
			Notes:         "",
			Events:        0,
			RSIMember:     true,
		}

		// get the user's primary org, if the nickname is an RSI handle
		reg := regexp.MustCompile(`\[(.*?)\] `)
		trueNick := reg.ReplaceAllString(nick, "")
		po, rank, err := rsi.GetOrgInfo(trueNick)
		if err != nil {
			if !errors.Is(err, rsi.UserNotFound) {
				return errors.Wrap(err, "getting rsi based rank")
			}

			u.RSIMember = false
		}
		if member.User.Bot {
			rank = users.Bot
		}
		u.Rank = rank
		u.PrimaryOrg = po

		for _, su := range storedUsers {
			if member.User.ID == su.Id {
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
	members, err := bot.GuildMembers(b.GuildId, "", 1000)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild members")
	}

	return members, nil
}

func (b *Bot) GetMember(id string) (*discordgo.Member, error) {
	member, err := bot.GuildMember(b.GuildId, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild member")
	}

	return member, nil
}
