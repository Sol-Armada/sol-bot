package bot

import (
	"context"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
	"golang.org/x/exp/slices"
)

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
				rateBucket := b.Ratelimiter.GetBucket("guild_member_check")
				if rateBucket.Remaining == 0 {
					log.Warn("hit a rate limit. relaxing until it goes away")
					time.Sleep(b.Ratelimiter.GetWaitTime(rateBucket, 0))
					continue
				}

				// get the discord members
				m, err := b.GetMembers()
				if err != nil {
					log.WithError(err).Error("bot getting members")
					return
				}

				// get the stored members
				storedUsers := []*user.User{}
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
	members, err := b.GuildMembers(b.GuildId, "", 1000)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild members")
	}

	return members, nil
}

func (b *Bot) GetMember(id string) (*discordgo.Member, error) {
	member, err := b.GuildMember(b.GuildId, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild member")
	}

	return member, nil
}

func updateMembers(m []*discordgo.Member, storedUsers []*user.User) error {
	log.Debug("checking users")

	for _, member := range m {
		time.Sleep(500 * time.Millisecond)

		u := user.New(member)

		for _, su := range storedUsers {
			if member.User.ID == su.ID {
				u.Events = su.Events
				u.Notes = su.Notes
				break
			}
		}

		u.Avatar = member.Avatar

		// get the user's primary org, if the nickname is an RSI handle
		trueNick := u.GetTrueNick()
		po, ao, rank, err := rsi.GetOrgInfo(trueNick)
		if err != nil {
			if !errors.Is(err, rsi.UserNotFound) {
				return errors.Wrap(err, "getting rsi based rank")
			}

			u.RSIMember = false
		}
		if slices.Contains(member.Roles, config.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
			rank = ranks.Recruit
		}
		if member.User.Bot {
			rank = ranks.Bot
		}
		for _, a := range config.GetStringSlice("allies") {
			if strings.EqualFold(u.PrimaryOrg, a) {
				rank = ranks.Ally
				break
			}
		}
		u.Rank = rank

		u.PrimaryOrg = po

		for _, affiliatedOrg := range ao {
			if slices.Contains(config.GetStringSlice("enemies"), affiliatedOrg) {
				u.BadAffiliation = true
				goto SAVE
			}
		}

	SAVE:
		if err := u.Save(); err != nil {
			return errors.Wrap(err, "saving new user")
		}
	}

	return nil
}

func cleanMembers(m []*discordgo.Member, storedUsers []*user.User) error {
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
