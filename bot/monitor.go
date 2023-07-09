package bot

import (
	"context"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/health"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/exp/slices"
)

func (b *Bot) UserMonitor(stop <-chan bool, done chan bool) {
	logger := log.WithField("func", "UserMonitor")
	logger.Info("monitoring discord for users")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastChecked := time.Now().Add(-30 * time.Minute)
	for {
		select {
		case <-stop:
			logger.Info("stopping monitor")
			goto DONE
		case <-ticker.C:
			if !health.IsHealthy() {
				time.Sleep(10 * time.Second)
				continue
			}
			if time.Now().After(lastChecked.Add(30 * time.Minute)) {
				logger.Info("scanning users")
				if stores.Storage == nil {
					logger.Debug("storage not setup, waiting a bit")
					time.Sleep(10 * time.Second)
					continue
				}

				// rate limit protection
				rateBucket := b.Ratelimiter.GetBucket("guild_member_check")
				if rateBucket.Remaining == 0 {
					logger.Warn("hit a rate limit. relaxing until it goes away")
					time.Sleep(b.Ratelimiter.GetWaitTime(rateBucket, 0))
					continue
				}

				// get the discord members
				m, err := b.GetMembers()
				if err != nil {
					logger.WithError(err).Error("bot getting members")
					return
				}

				// get the stored members
				storedUsers := []*user.User{}
				cur, err := stores.Storage.GetUsers(bson.M{"updated": bson.M{"$lte": time.Now().Add(-30 * time.Minute).UTC()}})
				if err != nil {
					logger.WithError(err).Error("getting users for updating")
					return
				}
				if err := cur.All(context.Background(), &storedUsers); err != nil {
					logger.WithError(err).Error("getting users from collection for update")
					return
				}

				// actually do the members update
				if err := updateMembers(m); err != nil {
					if strings.Contains(err.Error(), "Forbidden") {
						lastChecked = time.Now()
						continue
					}

					logger.WithError(err).Error("updating members")
					return
				}

				// do some cleaning
				if err := cleanMembers(m, storedUsers); err != nil {
					logger.WithError(err).Error("cleaning up the members")
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

func updateMembers(m []*discordgo.Member) error {
	log.WithFields(log.Fields{
		"discord_members": len(m),
	}).Debug("checking users")

	for _, member := range m {
		time.Sleep(1 * time.Second)

		// get the stord user, if we have one
		u, err := user.Get(member.User.ID)
		if err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				log.WithError(err).Error("getting user for update")
				continue
			}

			u = user.New(member)
		}
		u.Discord = member
		u.Name = u.GetTrueNick()

		// rsi related stuff
		u, err = rsi.GetOrgInfo(u)
		if err != nil {
			if strings.Contains(err.Error(), "Forbidden") {
				log.Warn("rsi limit")
				continue
			}

			if !errors.Is(err, rsi.UserNotFound) {
				return errors.Wrap(err, "getting rsi based rank")
			}

			log.WithField("user", u).Debug("user not found")
			u.RSIMember = false
		}

		// discord related stuff
		u.Avatar = member.Avatar
		if slices.Contains(member.Roles, config.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
			u.Rank = ranks.Recruit
		}
		if member.User.Bot {
			u.IsBot = true
		}

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
