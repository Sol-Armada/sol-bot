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
	ticker := time.NewTicker(30 * time.Second) // Reduced frequency when idle
	defer ticker.Stop()

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

			storedMembers, err := members.List(0)
			if err != nil {
				logger.Error("getting stored members", "error", err)
				continue
			}

			// do some cleaning
			// Create a map of Discord member IDs for efficient lookup
			discordMemberMap := make(map[string]bool, len(discordMembers))
			for _, discordMember := range discordMembers {
				discordMemberMap[discordMember.User.ID] = true
			}

			for _, storedMember := range storedMembers {
				select {
				case <-stop:
					logger.Warn("stopping monitor")
					return
				default:
				}

				if !discordMemberMap[storedMember.Id] || storedMember.IsBot {
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
	defer func() {
		if err := bot.UpdateCustomStatus(""); err != nil {
			logger.Error("updating custom status", "error", err)
		}
	}()
	logger.Debug("checking members", "discord_members", len(discordMembers))

	// Cache role IDs for efficiency
	recruitRoleID := settings.GetString("DISCORD.ROLE_IDS.RECRUIT")
	allyRoleID := settings.GetString("DISCORD.ROLE_IDS.ALLY")

	// Collect members to save for batch processing
	var membersToSave []members.Member
	var processingErrors []error

	logger.Info(fmt.Sprintf("updating %d members", len(discordMembers)))
	if err := bot.UpdateCustomStatus("Updating members..."); err != nil {
		logger.Error("updating custom status", "error", err)
	}
	wait := 0
	for i, discordMember := range discordMembers {
		select {
		case <-stop:
			logger.Warn("stopping monitor")
			return nil
		default:
			if wait > 0 {
				wait--
				time.Sleep(1 * time.Second)
				continue
			}
		}

		time.Sleep(time.Second)
		if (i > 0 && i%10 == 0) || i > len(discordMembers)-10 {
			if err := bot.UpdateCustomStatus(fmt.Sprintf("Updating members... (%d/%d)", i, len(discordMembers))); err != nil {
				logger.Error("updating custom status", "error", err)
			}
		}

		mlogger := logger.With(
			"id", discordMember.User.ID,
			"name", discordMember.DisplayName())

		mlogger.Debug("updating member")

		if discordMember.User.Bot {
			mlogger.Debug("skipping bot")
			continue
		}

		// get the stored user, if we have one
		member, err := members.Get(discordMember.User.ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				mlogger.Error("getting member for update", "error", err)
				processingErrors = append(processingErrors, err)
				continue
			}

			member = members.New(discordMember)
		}

		member.Name = strings.ReplaceAll(member.GetTrueNick(discordMember), ".", "")
		if member.Joined.IsZero() {
			member.Joined = discordMember.JoinedAt.UTC()
		}

		// rsi related stuff
		if err = rsi.UpdateRsiInfo(member); err != nil {
			if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "Bad Gateway") {
				mlogger.Warn("rsi service issue... waiting for rate limit")
				if err := bot.UpdateCustomStatus("Updating members... waiting for RSI rate limit"); err != nil {
					logger.Error("updating custom status", "error", err)
				}
				wait = 10
				continue
			}

			if strings.Contains(err.Error(), "context deadline exceeded") {
				mlogger.Warn("getting rsi info", "error", err)
				continue
			}

			if !errors.Is(err, rsi.RsiUserNotFound) {
				processingErrors = append(processingErrors, errors.Wrap(err, "getting rsi info"))
				continue
			}

			mlogger.Debug("rsi user not found", "error", err)
			member.RSIMember = false
		}

		// Create role map for efficient lookup
		roleMap := make(map[string]bool, len(discordMember.Roles))
		for _, roleID := range discordMember.Roles {
			roleMap[roleID] = true
		}

		// discord related stuff
		member.Avatar = discordMember.User.Avatar
		if roleMap[recruitRoleID] {
			mlogger.Debug("is recruit")
			member.Rank = ranks.Recruit
			member.IsAffiliate = false
			member.IsAlly = false
			member.IsGuest = false
		}
		if roleMap[allyRoleID] {
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

		// Add to batch for saving
		membersToSave = append(membersToSave, *member)
	}

	// Save members in batch (if the members package supports it) or individually with error collection
	for _, member := range membersToSave {
		if err := member.Save(); err != nil {
			logger.Error("saving member", "error", err, "member_id", member.Id)
			processingErrors = append(processingErrors, err)
		}
	}

	// Log any processing errors but don't fail the entire operation
	if len(processingErrors) > 0 {
		logger.Warn("encountered errors during member processing", "error_count", len(processingErrors))
	}

	return nil
}
