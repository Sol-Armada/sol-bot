package bot

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/cache"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/health"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/rsi"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/exp/slices"
)

func (b *Bot) UserMonitor(stop <-chan bool, done chan bool) {
	logger := log.WithField("func", "UserMonitor")
	logger.Info("monitoring discord for members")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastChecked := time.Now().Add(-30 * time.Minute)
	d := false
	for {
		select {
		case <-stop:
			logger.Warn("stopping monitor")
			d = true
			goto DONE
		case <-ticker.C:
			if !health.IsHealthy() {
				logger.Debug("not healthy")
				time.Sleep(10 * time.Second)
				continue
			}
			if time.Now().After(lastChecked.Add(30 * time.Minute)) {
				logger.Info("scanning members")
				if !stores.Connected() {
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
					continue
				}

				// actually do the members update
				if err := updateMembers(m); err != nil {
					if strings.Contains(err.Error(), "Forbidden") {
						lastChecked = time.Now()
						continue
					}

					logger.WithError(err).Error("updating members")
					continue
				}

				// get the stored members
				storedUsers := []*members.Member{}
				rawUsers := cache.Cache.GetUsers()
				for _, v := range rawUsers {
					uByte, _ := json.Marshal(v)
					u := members.Member{}
					if err := json.Unmarshal(uByte, &u); err != nil {
						logger.WithError(err).Error("unmarshalling user from cache")
						continue
					}
					storedUsers = append(storedUsers, &u)
				}

				// do some cleaning
				if err := cleanMembers(m, storedUsers); err != nil {
					logger.WithError(err).Error("cleaning up the members")
					continue
				}

				lastChecked = time.Now()
			}

			continue
		}
	DONE:
		if d {
			done <- true
			return
		}
	}
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

func updateMembers(discordMembers []*discordgo.Member) error {
	logger := log.WithField("func", "updateMembers")
	logger.WithFields(log.Fields{
		"discord_members": len(discordMembers),
	}).Debug("checking members")

	logger.Debugf("updating %d members", len(discordMembers))
	for _, discordMember := range discordMembers {
		time.Sleep(1 * time.Second)
		mlogger := logger.WithField("member", discordMember)
		mlogger.Debug("updating member")

		// get the stord user, if we have one
		member, err := members.Get(discordMember.User.ID)
		if err != nil && !errors.Is(err, members.MemberNotFound) {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				mlogger.WithError(err).Error("getting member for update")
				continue
			}

			member = members.New(discordMember)
		}
		if member == nil {
			member = members.New(discordMember)
		}
		member.Name = strings.ReplaceAll(member.GetTrueNick(discordMember), ".", "")
		member.RSIMember = true

		// rsi related stuff
		member, err = rsi.UpdateRsiInfo(member)
		if err != nil {
			if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "Bad Gateway") {
				return err
			}

			if !errors.Is(err, rsi.UserNotFound) {
				return errors.Wrap(err, "getting rsi based rank")
			}

			mlogger.WithField("user", member).Debug("user not found")
			member.RSIMember = false
		}

		if member.RSIMember {
			member.BadAffiliation = false
			member.IsAlly = false

			for _, affiliatedOrg := range member.Affilations {
				if slices.Contains(config.GetStringSlice("enemies"), affiliatedOrg) {
					member.BadAffiliation = true
					break
				}
			}
			for _, ally := range config.GetStringSlice("allies") {
				if strings.EqualFold(member.PrimaryOrg, ally) {
					member.IsAlly = true
					break
				}
			}
		}

		// discord related stuff
		member.Avatar = discordMember.Avatar
		if slices.Contains(discordMember.Roles, config.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
			mlogger.Debug("is recruit")
			member.Rank = ranks.Recruit
		}
		if discordMember.User.Bot {
			member.IsBot = true
		}

		// fill legacy
		member.LegacyEvents = member.Events

		if err := member.Save(); err != nil {
			return err
		}
	}

	return nil
}

func cleanMembers(discordMembers []*discordgo.Member, storedMembers []*members.Member) error {
	for _, member := range storedMembers {
		for _, discordMember := range discordMembers {
			if member.Id == discordMember.User.ID {
				goto CONTINUE
			}
		}

		log.WithField("member", member).Info("deleting member")
		if err := member.Delete(); err != nil {
			return errors.Wrap(err, "cleaning members")
		}
	CONTINUE:
		continue
	}

	return nil
}
