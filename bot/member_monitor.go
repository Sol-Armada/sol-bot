package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
)

var logger *slog.Logger

func MemberMonitor(ctx context.Context, logger *slog.Logger) error {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("panic in member monitor", "error", err)
		}
	}()

	logger.Debug("MemberMonitor starting", "bot_nil", bot == nil)
	if bot == nil {
		logger.Error("bot instance is nil")
		return errors.New("bot instance is nil")
	}

	logger.Debug("bot instance validated", "guild_id", bot.GuildId, "session_nil", bot.Session == nil)

	if !health.IsHealthy() {
		logger.Debug("not healthy")
		time.Sleep(10 * time.Second)
		return errors.New("not healthy")
	}

	start := time.Now().UTC()
	logger.Info("scanning members")

TRY_AGAIN:
	// rate limit protection
	if bot.Ratelimiter != nil {
		rateBucket := bot.Ratelimiter.GetBucket("guild_member_check")
		if rateBucket != nil && rateBucket.Remaining == 0 {
			logger.Warn("hit a rate limit. relaxing until it goes away")
			time.Sleep(bot.Ratelimiter.GetWaitTime(rateBucket, 0))
			goto TRY_AGAIN
		}
	}

	// get the discord members
	logger.Debug("calling bot.GetDiscordMembers()")

	// Double-check bot is not nil before calling
	if bot == nil {
		logger.Error("bot became nil between checks")
		return errors.New("bot became nil between checks")
	}

	logger.Debug("about to call GetDiscordMembers", "bot_nil", bot == nil)

	// Add extra safety check and debug info
	if bot == nil {
		logger.Error("bot is nil right before GetDiscordMembers call")
		return errors.New("bot is nil")
	}

	logger.Debug("calling bot.GetDiscordMembers() - bot is not nil")
	discordMembers, err := bot.GetDiscordMembers(logger)
	logger.Debug("GetDiscordMembers call returned", "error", err != nil)

	if err != nil {
		logger.Error("GetDiscordMembers failed", "error", err)
		return errors.Wrap(err, "getting discord members")
	}

	logger.Info(fmt.Sprintf("got %d discord members", len(discordMembers)))
	logger.Debug("discord members retrieved", "count", len(discordMembers), "first_member_nil", len(discordMembers) > 0 && discordMembers[0] == nil)

	// actually do the members update
	if err := updateMembers(ctx, logger, discordMembers); err != nil {
		if strings.Contains(err.Error(), "Forbidden") {
			logger.Warn("discord service issue... waiting for rate limit")
			time.Sleep(5 * time.Minute)
			goto TRY_AGAIN
		}

		return errors.Wrap(err, "updating members")
	}

	logger.Debug("getting stored members")
	storedMembers, err := members.List(0)
	if err != nil {
		logger.Error("failed to get stored members", "error", err)
		return errors.Wrap(err, "getting stored members")
	}
	logger.Debug("retrieved stored members", "count", len(storedMembers))

	// do some cleaning
	// Create a map of Discord member IDs for efficient lookup
	logger.Debug("creating discord member map", "member_count", len(discordMembers))
	discordMemberMap := make(map[string]bool, len(discordMembers))
	nilMemberCount := 0
	nilUserCount := 0
	for i, discordMember := range discordMembers {
		if discordMember == nil {
			nilMemberCount++
			logger.Warn("nil discord member found", "index", i)
			continue
		}
		if discordMember.User == nil {
			nilUserCount++
			logger.Warn("nil discord member user found", "index", i, "member_id", "unknown")
			continue
		}
		discordMemberMap[discordMember.User.ID] = true
	}
	if nilMemberCount > 0 || nilUserCount > 0 {
		logger.Warn("found nil members during map creation", "nil_members", nilMemberCount, "nil_users", nilUserCount)
	}

	logger.Debug("cleaning up stored members")
	deletedCount := 0
	for i, storedMember := range storedMembers {
		memberLogger := logger.With("stored_member_id", storedMember.Id, "stored_member_name", storedMember.Name, "index", i)

		if !discordMemberMap[storedMember.Id] || storedMember.IsBot {
			memberLogger.Debug("deleting stored member", "reason", map[string]bool{
				"not_in_discord": !discordMemberMap[storedMember.Id],
				"is_bot":         storedMember.IsBot,
			})

			if err := storedMember.Delete(); err != nil {
				memberLogger.Error("deleting member", "error", err)
				continue
			}
			deletedCount++
			memberLogger.Debug("member deleted successfully")
		}
	}
	logger.Debug("cleanup completed", "deleted_count", deletedCount)

	logger.Info("members updated", "count", len(discordMembers), "duration", time.Since(start))
	return nil
}

func (b *Bot) GetDiscordMembers(logger *slog.Logger) ([]*discordgo.Member, error) {
	logger.Debug("GetDiscordMembers function entry")

	var guildID string
	if b != nil {
		guildID = b.GuildId
	}
	logger.Debug("GetDiscordMembers called", "bot_nil", b == nil, "session_nil", b != nil && b.Session == nil, "guild_id", guildID)

	if b == nil {
		logger.Error("bot instance is nil in GetDiscordMembers")
		return nil, errors.New("bot instance is nil")
	}
	if b.Session == nil {
		logger.Error("bot session is nil in GetDiscordMembers")
		return nil, errors.New("bot session is nil")
	}

	// Check Discord session state
	logger.Debug("discord session state",
		"state_ready", b.Session.State != nil,
		"token_set", b.Session.Token != "",
		"client_timeout", b.Session.Client.Timeout)

	if b.Session.State != nil {
		var userID string
		if b.Session.State.User != nil {
			userID = b.Session.State.User.ID
		}
		logger.Debug("session state details",
			"user_id", userID,
			"guilds_count", len(b.Session.State.Guilds))
	}

	// Try a simple guild lookup first to test connectivity
	logger.Debug("testing discord connection with guild lookup")
	guild, guildErr := b.Session.Guild(b.GuildId)
	if guildErr != nil {
		logger.Error("guild lookup failed - connection issues", "error", guildErr, "guild_id", b.GuildId)
		return nil, errors.Wrap(guildErr, "discord connection test failed")
	}
	logger.Debug("guild lookup successful", "guild_name", guild.Name, "member_count", guild.MemberCount)

	logger.Debug("about to call b.GuildMembers", "guild_id", b.GuildId)

	// Add defer to catch any panic in the Discord API call
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic in GuildMembers call", "panic", r, "guild_id", b.GuildId)
		}
	}()

	// Log the exact parameters being used
	logger.Debug("GuildMembers parameters", "guild_id", b.GuildId, "after", "", "limit", 1000)

	startTime := time.Now()
	members, err := b.GuildMembers(b.GuildId, "", 1000)
	duration := time.Since(startTime)

	logger.Debug("GuildMembers call completed",
		"duration_ms", duration.Milliseconds(),
		"error", err != nil,
		"error_msg", err,
		"members_nil", members == nil)

	if err != nil {
		logger.Error("GuildMembers call failed", "error", err, "guild_id", b.GuildId)
		return nil, errors.Wrap(err, "getting guild members")
	}

	logger.Debug("GuildMembers call successful", "member_count", len(members))
	return members, nil
}

func (b *Bot) GetMember(id string) (*discordgo.Member, error) {
	member, err := b.GuildMember(b.GuildId, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting guild member")
	}

	return member, nil
}

func updateMembers(ctx context.Context, logger *slog.Logger, discordMembers []*discordgo.Member) error {
	defer func() {
		removeStatusMessage("member_monitor")
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
		case <-ctx.Done():
			logger.Info("member update cancelled")
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
			upsertStatusMessage("member_monitor", fmt.Sprintf("Updating members... (%d/%d)", i, len(discordMembers)))
		}

		// Check for nil pointers before accessing User fields
		if discordMember == nil {
			logger.Warn("skipping nil discord member", "index", i)
			continue
		}
		if discordMember.User == nil {
			logger.Warn("skipping discord member with nil user", "index", i)
			continue
		}

		mlogger := logger.With(
			"id", discordMember.User.ID,
			"name", discordMember.DisplayName(),
			"index", i)

		mlogger.Debug("updating member")

		if discordMember.User.Bot {
			mlogger.Debug("skipping bot")
			continue
		}

		// get the stored user, if we have one
		mlogger.Debug("getting stored member")
		member, err := members.Get(discordMember.User.ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				mlogger.Error("getting member for update", "error", err)
				processingErrors = append(processingErrors, err)
				continue
			}

			mlogger.Debug("creating new member")
			member = members.New(discordMember)
			if member == nil {
				mlogger.Error("members.New returned nil")
				continue
			}
		}

		mlogger.Debug("member retrieved/created", "member_nil", member == nil)

		mlogger.Debug("updating member fields", "current_name", member.Name)
		truenick := member.GetTrueNick(discordMember)
		mlogger.Debug("got true nick", "truenick", truenick)
		member.Name = strings.ReplaceAll(truenick, ".", "")

		if member.Joined.IsZero() {
			mlogger.Debug("setting joined date", "joined_at", discordMember.JoinedAt)
			member.Joined = discordMember.JoinedAt.UTC()
		}

		// rsi related stuff
		mlogger.Debug("updating RSI info")
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
		mlogger.Debug("creating role map", "role_count", len(discordMember.Roles))
		roleMap := make(map[string]bool, len(discordMember.Roles))
		for _, roleID := range discordMember.Roles {
			if roleID == "" {
				mlogger.Warn("empty role ID found")
				continue
			}
			roleMap[roleID] = true
		}

		// discord related stuff
		mlogger.Debug("updating discord fields", "avatar", discordMember.User.Avatar)
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
		mlogger.Debug("adding member to save batch")
		membersToSave = append(membersToSave, *member)
	}

	// Save members in batch (if the members package supports it) or individually with error collection
	logger.Debug("saving members", "count", len(membersToSave))
	for i, member := range membersToSave {
		memberLogger := logger.With("member_id", member.Id, "member_name", member.Name, "batch_index", i)
		memberLogger.Debug("saving member")
		if err := member.Save(); err != nil {
			memberLogger.Error("saving member", "error", err)
			processingErrors = append(processingErrors, err)
		} else {
			memberLogger.Debug("member saved successfully")
		}
	}

	// Log any processing errors but don't fail the entire operation
	if len(processingErrors) > 0 {
		logger.Warn("encountered errors during member processing", "error_count", len(processingErrors))
	}

	return nil
}
