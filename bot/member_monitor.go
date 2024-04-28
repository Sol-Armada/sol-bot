package bot

import (
	"context"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

func MemberMonitor(stop <-chan bool) {
	logger := log.WithField("func", "UserMonitor")
	logger.Info("monitoring discord for members")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	membersStore, ok := stores.Get().GetMembersStore()
	if !ok {
		logger.Error("failed to get members store")
		return
	}

	lastChecked := time.Now().UTC().Add(-30 * time.Minute)
	for {
		select {
		case <-stop:
			logger.Warn("stopping monitor")
			return
		case <-ticker.C:
		}

		if !health.IsHealthy() {
			logger.Debug("not healthy")
			time.Sleep(10 * time.Second)
			continue
		}

		if time.Now().UTC().After(lastChecked.Add(30 * time.Minute)) {
			start := time.Now().UTC()
			logger.Info("scanning members")
			// TODO: Check if system is healthy

			// rate limit protection
			rateBucket := bot.Ratelimiter.GetBucket("guild_member_check")
			if rateBucket.Remaining == 0 {
				logger.Warn("hit a rate limit. relaxing until it goes away")
				time.Sleep(bot.Ratelimiter.GetWaitTime(rateBucket, 0))
				continue
			}

			// get the discord members
			discordMembers, err := bot.GetDiscordMembers()
			if err != nil {
				logger.WithError(err).Error("bot getting members")
				continue
			}

			// actually do the members update
			if err := updateMembers(discordMembers); err != nil {
				if strings.Contains(err.Error(), "Forbidden") {
					lastChecked = time.Now()
					continue
				}

				logger.WithError(err).Error("updating members")
				continue
			}

			// get the stored members
			storedMembers := []*members.Member{}
			cur, err := membersStore.List(bson.M{}, &options.FindOptions{
				Sort: bson.M{"rank": 1},
			})
			if err != nil {
				logger.WithError(err).Error("getting stored members")
				continue
			}

			if err := cur.All(context.Background(), &storedMembers); err != nil {
				logger.WithError(err).Error("reading in stored members")
				continue
			}

			// do some cleaning
			for _, storedMember := range storedMembers {
				if !stillInDiscord(storedMember, discordMembers) || storedMember.IsBot {
					if err := storedMember.Delete(); err != nil {
						logger.WithField("member", storedMember).WithError(err).Error("deleting member")
						continue
					}
				}
			}

			lastChecked = time.Now()

			logger.WithField("duration", time.Since(start)).Info("members updated")
		}

		continue
	}
}

func (b *Bot) UpdateMember() error {
	return nil
}

func (b *Bot) GetDiscordMembers() ([]*discordgo.Member, error) {
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

	logger.Infof("updating %d members", len(discordMembers))
	for _, discordMember := range discordMembers {
		time.Sleep(1 * time.Second)
		mlogger := logger.WithFields(log.Fields{
			"id":   discordMember.User.ID,
			"name": discordMember.DisplayName(),
		})
		mlogger.Info("updating member")

		if discordMember.User.Bot {
			mlogger.Debug("skipping bot")
			continue
		}

		// get the stord user, if we have one
		member, err := members.Get(discordMember.User.ID)
		if err != nil && !errors.Is(err, members.MemberNotFound) {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				mlogger.WithError(err).Error("getting member for update")
				continue
			}

			member = members.New(discordMember)
		}

		member.Name = strings.ReplaceAll(member.GetTrueNick(discordMember), ".", "")

		// rsi related stuff
		if err = rsi.UpdateRsiInfo(member); err != nil {
			if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "Bad Gateway") {
				return err
			}

			if strings.Contains(err.Error(), "context deadline exceeded") {
				mlogger.WithError(err).Warn("getting rsi info")
				continue
			}

			if !errors.Is(err, rsi.RsiUserNotFound) {
				return errors.Wrap(err, "getting rsi info")
			}

			mlogger.WithField("member", member).Debug("rsi user not found")
			member.RSIMember = false
		}

		// discord related stuff
		member.Avatar = discordMember.Avatar
		if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
			mlogger.Debug("is recruit")
			member.Rank = ranks.Recruit
		}
		if discordMember.User.Bot {
			member.IsBot = true
		}

		if err := member.Save(); err != nil {
			return err
		}

		// handle rank updates on members
		// rankRoles := settings.GetStringMapString("DISCORD.ROLES.RANKS")
		// membersRoleId := rankRoles[strings.ToLower(member.Rank.String())]
		// if (!utils.StringSliceContains(discordMember.Roles, membersRoleId) && !member.IsGuest) || member.Rank == ranks.Member {
		// 	if !member.IsGuest && !member.IsAlly && !member.IsAffiliate {
		// 		if member.Rank != ranks.Member {
		// 			logger.Debug("is not just a member, adding role: " + membersRoleId)
		// 			_ = bot.GuildMemberRoleAdd(bot.GuildId, member.Id, membersRoleId)
		// 		}
		// 		_ = bot.GuildMemberRoleAdd(bot.GuildId, member.Id, rankRoles["member"])
		// 	}

		// 	if member.IsAlly {
		// 		_ = bot.GuildMemberRoleAdd(bot.GuildId, member.Id, rankRoles["ally"])
		// 	}

		// 	if member.IsAffiliate {
		// 		_ = bot.GuildMemberRoleAdd(bot.GuildId, member.Id, rankRoles["affiliate"])
		// 	}

		// 	for rankName, rankId := range rankRoles {
		// 		// reasons not to remove a rank
		// 		// member  - all not guests and not recruits have member
		// 		// ally    - applied somewhere else
		// 		// affiliate - applied somewhere else
		// 		if rankName != strings.ToLower(member.Rank.String()) && rankName != "member" && rankName != "ally" && rankName != "affiliate" {
		// 			_ = bot.GuildMemberRoleRemove(bot.GuildId, member.Id, rankId)
		// 		}
		// 	}

		// 	nick := member.GetTrueNick(discordMember)
		// 	if !member.IsGuest && !member.IsAlly && !member.IsAffiliate && member.IsRanked() {
		// 		nick = "[" + member.Rank.ShortString() + "] " + nick
		// 		if member.Suffix != "" {
		// 			nick += " (" + member.Suffix + ")"
		// 		}
		// 	}

		// 	logger.WithField("nick", nick).Debug("setting nick")
		// 	if err = bot.GuildMemberNickname(bot.GuildId, member.Id, nick); err != nil {
		// 		logger.WithError(err).Error("setting nick")
		// 	}
		// }
	}

	return nil
}

func stillInDiscord(member *members.Member, discordMembers []*discordgo.Member) bool {
	for _, discordMember := range discordMembers {
		if member.Id == discordMember.User.ID {
			return true
		}
	}

	return false
}
