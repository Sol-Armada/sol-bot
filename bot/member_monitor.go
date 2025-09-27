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
	"github.com/sol-armada/sol-bot/utils"
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

	// Create exponential backoff for Discord API calls
	backoff := utils.NewExponentialBackoff(
		1*time.Second, // initial delay
		5*time.Minute, // max delay
		2.0,           // multiplier
		3,             // max retries
		logger,
	)

	var discordMembers []*discordgo.Member
	err := backoff.Execute(func() error {
		// rate limit protection
		if bot.Ratelimiter != nil {
			rateBucket := bot.Ratelimiter.GetBucket("guild_member_check")
			if rateBucket != nil && rateBucket.Remaining == 0 {
				logger.Warn("hit a rate limit. waiting for rate limit to reset")
				time.Sleep(bot.Ratelimiter.GetWaitTime(rateBucket, 0))
			}
		}

		// Double-check bot is not nil before calling
		if bot == nil {
			return errors.New("bot instance is nil")
		}

		logger.Debug("calling bot.GetDiscordMembers()")
		var fetchErr error
		discordMembers, fetchErr = bot.GetDiscordMembers(logger)
		if fetchErr != nil {
			if strings.Contains(fetchErr.Error(), "Forbidden") {
				logger.Warn("discord service issue - forbidden access")
				return fetchErr
			}
			return errors.Wrap(fetchErr, "getting discord members")
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to get discord members after retries")
	}

	logger.Info(fmt.Sprintf("got %d discord members", len(discordMembers)))

	// Validate and filter discord members
	validDiscordMembers := members.ValidateDiscordMembers(discordMembers)
	logger.Info(fmt.Sprintf("filtered to %d valid discord members", len(validDiscordMembers)))

	// actually do the members update
	if err := updateMembers(ctx, logger, validDiscordMembers); err != nil {
		return errors.Wrap(err, "updating members")
	}

	logger.Debug("getting stored member IDs for cleanup")
	storedMemberIDs, err := members.GetStoredMemberIDs()
	if err != nil {
		logger.Error("failed to get stored member IDs", "error", err)
		return errors.Wrap(err, "getting stored member IDs")
	}
	logger.Debug("retrieved stored member IDs", "count", len(storedMemberIDs))

	// do some cleaning - create a map of Discord member IDs for efficient lookup
	logger.Debug("creating discord member map", "member_count", len(validDiscordMembers))
	discordMemberMap := make(map[string]bool, len(validDiscordMembers))
	for _, discordMember := range validDiscordMembers {
		discordMemberMap[discordMember.User.ID] = true
	}

	logger.Debug("cleaning up stored members")
	deletedCount := 0
	for _, storedMemberID := range storedMemberIDs {
		if !discordMemberMap[storedMemberID] {
			memberLogger := logger.With("stored_member_id", storedMemberID)
			memberLogger.Debug("deleting stored member", "reason", "not_in_discord")

			// Get the member to check if it's a bot before deleting
			storedMember, getMemberErr := members.Get(storedMemberID)
			if getMemberErr != nil && !errors.Is(getMemberErr, members.MemberNotFound) {
				memberLogger.Error("getting member for deletion check", "error", getMemberErr)
				continue
			}

			// Skip deletion if member is not found (already deleted) or if it's a bot that should be kept
			if errors.Is(getMemberErr, members.MemberNotFound) {
				continue
			}

			if storedMember.IsBot {
				memberLogger.Debug("skipping bot member deletion")
				continue
			}

			if err := storedMember.Delete(); err != nil {
				memberLogger.Error("deleting member", "error", err)
				continue
			}
			deletedCount++
			memberLogger.Debug("member deleted successfully")
		}
	}
	logger.Debug("cleanup completed", "deleted_count", deletedCount)

	logger.Info("members updated", "count", len(validDiscordMembers), "duration", time.Since(start))
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

	if len(discordMembers) == 0 {
		logger.Info("no members to process")
		return nil
	}

	// Cache role IDs for efficiency - moved outside loop
	recruitRoleID := settings.GetString("DISCORD.ROLE_IDS.RECRUIT")
	allyRoleID := settings.GetString("DISCORD.ROLE_IDS.ALLY")

	// Track processing errors
	var processingErrors []error

	// Create RSI backoff for rate limiting
	rsiBackoff := utils.NewExponentialBackoff(
		1*time.Second,  // initial delay
		30*time.Second, // max delay for RSI calls
		2.0,            // multiplier
		2,              // max retries for RSI
		logger,
	)

	logger.Info(fmt.Sprintf("updating %d members", len(discordMembers)))
	if err := bot.UpdateCustomStatus("Updating members..."); err != nil {
		logger.Error("updating custom status", "error", err)
	}

	// Process members in chunks to reduce memory usage and improve performance
	const chunkSize = 50
	for chunkStart := 0; chunkStart < len(discordMembers); chunkStart += chunkSize {
		select {
		case <-ctx.Done():
			logger.Info("member update cancelled")
			return nil
		default:
		}

		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(discordMembers) {
			chunkEnd = len(discordMembers)
		}

		chunk := discordMembers[chunkStart:chunkEnd]
		chunkMembersToSave := make([]members.Member, 0, len(chunk))

		logger.Debug("processing chunk", "start", chunkStart, "end", chunkEnd, "size", len(chunk))
		upsertStatusMessage("member_monitor", fmt.Sprintf("Updating members... (%d/%d)", chunkEnd, len(discordMembers)))

		for i, discordMember := range chunk {
			globalIndex := chunkStart + i

			mlogger := logger.With(
				"id", discordMember.User.ID,
				"name", discordMember.DisplayName(),
				"index", globalIndex)

			mlogger.Debug("updating member")

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

			// Update basic member fields
			mlogger.Debug("updating member fields", "current_name", member.Name)
			truenick := member.GetTrueNick(discordMember)
			mlogger.Debug("got true nick", "truenick", truenick)
			member.Name = strings.ReplaceAll(truenick, ".", "")

			if member.Joined.IsZero() {
				mlogger.Debug("setting joined date", "joined_at", discordMember.JoinedAt)
				member.Joined = discordMember.JoinedAt.UTC()
			}

			// Update avatar
			member.Avatar = discordMember.User.Avatar

			// Process roles efficiently - create role map once
			roleMap := createRoleMap(discordMember.Roles, mlogger)
			updateMemberRoles(member, roleMap, recruitRoleID, allyRoleID, mlogger)

			// RSI related stuff with better error handling
			mlogger.Debug("updating RSI info")
			err = rsiBackoff.Execute(func() error {
				return rsi.UpdateRsiInfo(member)
			})

			if err != nil {
				if errors.Is(err, rsi.RsiUserNotFound) {
					mlogger.Debug("rsi user not found", "error", err)
					member.RSIMember = false
				} else {
					mlogger.Warn("failed to update RSI info after retries", "error", err)
					// Don't fail the entire operation for RSI issues
					member.RSIMember = false
				}
			}

			// Add to chunk batch for saving
			mlogger.Debug("adding member to chunk save batch")
			chunkMembersToSave = append(chunkMembersToSave, *member)
		}

		// Save chunk in batch
		if len(chunkMembersToSave) > 0 {
			logger.Debug("saving chunk", "count", len(chunkMembersToSave))
			if err := members.BulkSave(chunkMembersToSave); err != nil {
				logger.Error("bulk saving chunk", "error", err)
				processingErrors = append(processingErrors, err)
			} else {
				logger.Debug("chunk saved successfully", "count", len(chunkMembersToSave))
			}
		}
	}

	// Log any processing errors but don't fail the entire operation
	if len(processingErrors) > 0 {
		logger.Warn("encountered errors during member processing", "error_count", len(processingErrors))
	}

	return nil
}

// createRoleMap creates an efficient map for role lookups
func createRoleMap(roles []string, logger *slog.Logger) map[string]bool {
	roleMap := make(map[string]bool, len(roles))
	for _, roleID := range roles {
		if roleID == "" {
			logger.Warn("empty role ID found")
			continue
		}
		roleMap[roleID] = true
	}
	return roleMap
}

// updateMemberRoles efficiently updates member roles based on the role map
func updateMemberRoles(member *members.Member, roleMap map[string]bool, recruitRoleID, allyRoleID string, logger *slog.Logger) {
	// Reset role flags
	member.IsAffiliate = false
	member.IsAlly = false
	member.IsGuest = false
	member.IsBot = false

	if roleMap[recruitRoleID] {
		logger.Debug("is recruit")
		member.Rank = ranks.Recruit
		member.IsGuest = false
	} else if roleMap[allyRoleID] {
		logger.Debug("is ally")
		member.Rank = ranks.None
		member.IsAlly = true
		member.IsGuest = false
	} else {
		// Default to guest if no specific roles found
		member.IsGuest = true
	}
}
