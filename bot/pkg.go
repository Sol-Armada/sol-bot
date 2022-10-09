package bot

import (
	"fmt"
	"regexp"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/users"
)

type Bot struct {
	GuildId string

	*discordgo.Session
}

var bot *Bot

func New() (*Bot, error) {
	if bot == nil {
		log.Info("starting new discord bot")
		b, err := discordgo.New(fmt.Sprintf("Bot %s", config.GetString("DISCORD.BOT_TOKEN")))
		if err != nil {
			return nil, err
		}

		bot = &Bot{
			config.GetString("DISCORD.GUILD_ID"),
			b,
		}

		if _, err := bot.Guild(config.GetString("DISCORD.GUILD_ID")); err != nil {
			return nil, err
		}
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
		log.Debug("checking users")

		m, err := b.GetMembers()
		if err != nil {
			log.WithError(err).Error("bot getting members")
			continue
		}
		store := users.GetStorage()
		storedUsers, err := store.GetUsers()
		if err != nil {
			log.WithError(err).Error("getting users for updating")
			continue
		}

		for _, member := range m {
			nick := member.User.Username
			if member.Nick != "" {
				nick = member.Nick
			}
			rank := users.Recruit
			if member.User.Bot {
				rank = users.Bot
			}

			u := &users.User{
				Nick:          nick,
				Id:            member.User.ID,
				Username:      member.User.Username,
				Discriminator: member.User.Discriminator,
				Avatar:        member.User.Avatar,
				Rank:          rank,
				Ally:          false,
				PrimaryOrg:    "",
				Notes:         "",
				Events:        0,
				RSIMember:     true,
			}

			// get the user's primary org, if the nickname is an RSI handle
			reg := regexp.MustCompile(`\[(.*?)\] `)
			trueNick := reg.ReplaceAllString(nick, "")
			po, err := rsi.GetPrimaryOrg(trueNick)
			if err != nil {
				if !errors.Is(err, rsi.UserNotFound) {
					log.WithError(err).Error("getting primary org")
					continue
				}
				u.RSIMember = false
			}
			u.PrimaryOrg = po

			for _, su := range storedUsers {
				if member.User.ID == su.Id {
					u.Events = su.Events
					u.Notes = su.Notes
					u.Ally = su.Ally
					u.Rank = su.Rank
					break
				}
			}

			if err := u.Save(); err != nil {
				log.WithError(err).Error("saving new user")
			}
		}

		time.Sleep(1 * time.Hour)
	}
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
