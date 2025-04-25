package bot

import (
	"fmt"
	"log/slog"
	"os"
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
	"golang.org/x/exp/slices"
)

var logger *slog.Logger

func MemberMonitor(stop <-chan bool) {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if settings.GetBool("LOG.DEBUG") {
		opts.Level = slog.LevelDebug
		slog.Debug("debug mode on")
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, opts))

	if !settings.GetBool("LOG.CLI") {
		f, err := os.OpenFile(settings.GetStringWithDefault("LOG.MEMBER_MONITOR_FILE", "/var/log/solbot/mm.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err.Error())
			os.Exit(1)
		}
		logger = slog.New(slog.NewJSONHandler(f, opts))
	}

	logger.Info("monitoring discord for members")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// membersStore, ok := stores.Get().GetMembersStore()
	// if !ok {
	// 	logger.Error("failed to get members store")
	// 	return
	// }

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
				logger.Error("bot getting members", "error", err)
				continue
			}

			// actually do the members update
			if err := updateMembers(discordMembers, stop); err != nil {
				if strings.Contains(err.Error(), "Forbidden") {
					lastChecked = time.Now()
					continue
				}

				logger.Error("updating members", "error", err)
				continue
			}

			// get the stored members
			// storedMembers := []*members.Member{}
			// cur, err := membersStore.List(bson.M{}, 0, 0)
			// if err != nil {
			// 	logger.Error("getting stored members", "error", err)
			// 	continue
			// }

			// if err := cur.All(context.Background(), &storedMembers); err != nil {
			// 	logger.Error("reading in stored members", "error", err)
			// 	continue
			// }
			storedMembers, err := members.List(0)
			if err != nil {
				logger.Error("getting stored members", "error", err)
				continue
			}

			// do some cleaning
			for _, storedMember := range storedMembers {
				select {
				case <-stop:
					logger.Warn("stopping monitor")
					return
				default:
				}

				if !stillInDiscord(storedMember, discordMembers) || storedMember.IsBot {
					if err := storedMember.Delete(); err != nil {
						logger.Error("deleting member", "error", err, "member", storedMember)
						continue
					}
				}
			}

			lastChecked = time.Now()

			logger.Info("members updated", "count", len(discordMembers), "duration", time.Since(start))
		}

		continue
	}
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

func updateMembers(discordMembers []*discordgo.Member, stop <-chan bool) error {
	logger.Debug("checking members", "discord_members", len(discordMembers))

	logger.Info(fmt.Sprintf("updating %d members", len(discordMembers)))
	for _, discordMember := range discordMembers {
		select {
		case <-stop:
			logger.Warn("stopping monitor")
			return nil
		default:
		}

		time.Sleep(1 * time.Second)
		mlogger := logger.With(
			"id", discordMember.User.ID,
			"name", discordMember.DisplayName())

		mlogger.Info("updating member")

		if discordMember.User.Bot {
			mlogger.Debug("skipping bot")
			continue
		}

		// get the stord user, if we have one
		member, err := members.Get(discordMember.User.ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				mlogger.Error("getting member for update", "error", err)
				continue
			}

			member = members.New(discordMember)
		}

		member.Name = strings.ReplaceAll(member.GetTrueNick(discordMember), ".", "")
		member.Joined = discordMember.JoinedAt.UTC()

		// rsi related stuff
		if err = rsi.UpdateRsiInfo(member); err != nil {
			if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "Bad Gateway") {
				return err
			}

			if strings.Contains(err.Error(), "context deadline exceeded") {
				mlogger.Warn("getting rsi info", "error", err)
				continue
			}

			if !errors.Is(err, rsi.RsiUserNotFound) {
				return errors.Wrap(err, "getting rsi info")
			}

			mlogger.Debug("rsi user not found", "error", err)
			member.RSIMember = false
		}

		// discord related stuff
		member.Avatar = discordMember.User.Avatar
		if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
			mlogger.Debug("is recruit")
			member.Rank = ranks.Recruit
			member.IsAffiliate = false
			member.IsAlly = false
			member.IsGuest = false
		}
		if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.ALLY")) {
			mlogger.Debug("is ally")
			member.Rank = ranks.None
			member.IsAffiliate = false
			member.IsAlly = true
			member.IsGuest = false
		}
		if discordMember.User.Bot {
			logger.Debug("is bot")
			member.Rank = ranks.None
			member.IsAffiliate = false
			member.IsAlly = false
			member.IsGuest = false
			member.IsBot = true
		}

		logger.Debug("updating member", "member", member)
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

func stillInDiscord(member members.Member, discordMembers []*discordgo.Member) bool {
	for _, discordMember := range discordMembers {
		if member.Id == discordMember.User.ID {
			return true
		}
	}

	return false
}
