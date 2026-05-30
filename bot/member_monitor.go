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
	"github.com/sol-armada/sol-bot/utils"
)

var logger *slog.Logger

const (
	memberProcessingChunkSize = 10
	discordAPIRetries         = 3

	discordAPIMaxDelay = 5 * time.Minute
	healthCheckDelay   = 10 * time.Second
)

// MemberMonitor is the main function that orchestrates the member monitoring process.
// It fetches Discord members, validates them, updates their information in the database,
// and cleans up members who are no longer in the Discord server.
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
		time.Sleep(healthCheckDelay)
		return errors.New("not healthy")
	}

	start := time.Now().UTC()
	logger.Info("scanning members")

	// Fetch and validate Discord members
	validDiscordMembers, err := fetchAndValidateDiscordMembers(logger)
	if err != nil {
		return errors.Wrap(err, "failed to fetch discord members")
	}

	// Update members
	if err := updateMembers(ctx, logger, validDiscordMembers); err != nil {
		return errors.Wrap(err, "updating members")
	}

	// Clean up members no longer in Discord
	if err := cleanupRemovedMembers(ctx, logger, validDiscordMembers); err != nil {
		return errors.Wrap(err, "cleaning up removed members")
	}

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

func updateMembers(ctx context.Context, logger *slog.Logger, discordMembers []*discordgo.Member) error {
	defer func() {
		removeStatusMessage("member_monitor")
	}()
	logger.Debug("checking members", "discord_members", len(discordMembers))

	if len(discordMembers) == 0 {
		logger.Info("no members to process")
		return nil
	}

	var processingErrors []error

	logger.Info(fmt.Sprintf("updating %d members", len(discordMembers)))
	if err := bot.UpdateCustomStatus("Updating members..."); err != nil {
		logger.Error("updating custom status", "error", err)
	}

	for chunkStart := 0; chunkStart < len(discordMembers); chunkStart += memberProcessingChunkSize {
		select {
		case <-ctx.Done():
			logger.Info("member update cancelled")
			return nil
		default:
		}

		chunkEnd := min(chunkStart+memberProcessingChunkSize, len(discordMembers))

		chunk := discordMembers[chunkStart:chunkEnd]

		logger.Debug("processing chunk", "start", chunkStart, "end", chunkEnd, "size", len(chunk))
		upsertStatusMessage("member_monitor", fmt.Sprintf("Updating members... (%d/%d)", chunkEnd, len(discordMembers)))

		// Process each member in the chunk
		processedMembers := processChunkMembers(ctx, chunk, chunkStart, logger, &processingErrors)

		// Save chunk in batch
		if len(processedMembers) > 0 {
			logger.Debug("saving chunk", "count", len(processedMembers))
			if err := members.BulkSave(processedMembers); err != nil {
				logger.Error("bulk saving chunk", "error", err)
				processingErrors = append(processingErrors, err)
				continue
			}
			logger.Debug("chunk saved successfully", "count", len(processedMembers))
		}
	}

	if len(processingErrors) > 0 {
		logger.Warn("encountered errors during member processing", "error_count", len(processingErrors))
	}

	return nil
}

// fetchAndValidateDiscordMembers fetches Discord members with retry logic and validates them
func fetchAndValidateDiscordMembers(logger *slog.Logger) ([]*discordgo.Member, error) {
	backoff := utils.NewExponentialBackoff(
		1*time.Second,      // initial delay
		discordAPIMaxDelay, // max delay
		2.0,                // multiplier
		discordAPIRetries,  // max retries
		logger,
	)

	var discordMembers []*discordgo.Member
	err := backoff.Execute(func() error {
		return fetchDiscordMembersWithRateLimit(logger, &discordMembers)
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get discord members after retries")
	}

	logger.Info("discord members fetched", "count", len(discordMembers))

	// Validate and filter discord members
	validMembers := members.ValidateDiscordMembers(discordMembers)
	logger.Info("discord members validated", "valid_count", len(validMembers), "filtered_out", len(discordMembers)-len(validMembers))

	return validMembers, nil
}

// fetchDiscordMembersWithRateLimit handles rate limiting and API calls
func fetchDiscordMembersWithRateLimit(logger *slog.Logger, discordMembers *[]*discordgo.Member) error {
	if bot.Ratelimiter != nil {
		rateBucket := bot.Ratelimiter.GetBucket("guild_member_check")
		if rateBucket != nil && rateBucket.Remaining == 0 {
			logger.Warn("hit a rate limit. waiting for rate limit to reset")
			time.Sleep(bot.Ratelimiter.GetWaitTime(rateBucket, 0))
		}
	}

	if bot == nil {
		return errors.New("bot instance is nil")
	}

	logger.Debug("calling bot.GetDiscordMembers()")
	var fetchErr error
	*discordMembers, fetchErr = bot.GetDiscordMembers(logger)
	if fetchErr != nil {
		if strings.Contains(fetchErr.Error(), "Forbidden") {
			logger.Warn("discord service issue - forbidden access")
			return fetchErr
		}
		return errors.Wrap(fetchErr, "getting discord members")
	}

	return nil
}

// cleanupRemovedMembers removes stored members that are no longer in Discord
func cleanupRemovedMembers(ctx context.Context, logger *slog.Logger, validDiscordMembers []*discordgo.Member) error {
	logger.Debug("getting stored member IDs for cleanup")
	storedMemberIDs, err := members.GetStoredMemberIDs()
	if err != nil {
		logger.Error("failed to get stored member IDs", "error", err)
		return errors.Wrap(err, "getting stored member IDs")
	}
	logger.Debug("retrieved stored member IDs", "count", len(storedMemberIDs))

	// Create a map of Discord member IDs for efficient lookup
	discordMemberMap := createDiscordMemberMap(validDiscordMembers)

	logger.Debug("cleaning up stored members")
	deletedCount := 0

	for _, storedMemberID := range storedMemberIDs {
		select {
		case <-ctx.Done():
			logger.Info("member cleanup cancelled")
			return nil
		default:
		}

		if !discordMemberMap[storedMemberID] {
			if deleted := deleteStoredMemberIfNeeded(logger, storedMemberID); deleted {
				deletedCount++
			}
		}
	}

	logger.Debug("cleanup completed", "deleted_count", deletedCount)
	return nil
}

// createDiscordMemberMap creates a map for efficient Discord member ID lookups
func createDiscordMemberMap(validDiscordMembers []*discordgo.Member) map[string]bool {
	discordMemberMap := make(map[string]bool, len(validDiscordMembers))
	for _, discordMember := range validDiscordMembers {
		discordMemberMap[discordMember.User.ID] = true
	}
	return discordMemberMap
}

// deleteStoredMemberIfNeeded deletes a stored member if it's safe to do so
func deleteStoredMemberIfNeeded(logger *slog.Logger, storedMemberID string) bool {
	memberLogger := logger.With("stored_member_id", storedMemberID)
	memberLogger.Debug("deleting stored member", "reason", "not_in_discord")

	// Get the member to check if it's a bot before deleting
	storedMember, getMemberErr := members.Get(storedMemberID)
	if getMemberErr != nil && !errors.Is(getMemberErr, members.MemberNotFound) {
		memberLogger.Error("getting member for deletion check", "error", getMemberErr)
		return false
	}

	// Skip deletion if member is not found (already deleted) or if it's a bot that should be kept
	if errors.Is(getMemberErr, members.MemberNotFound) {
		return false
	}

	if storedMember.IsBot {
		memberLogger.Debug("skipping bot member deletion")
		return false
	}

	if err := storedMember.Delete("Bot did not find on server"); err != nil {
		memberLogger.Error("deleting member", "error", err)
		return false
	}

	memberLogger.Debug("member deleted successfully")
	return true
}

// processChunkMembers processes a chunk of Discord members and returns the updated members
func processChunkMembers(
	ctx context.Context,
	chunk []*discordgo.Member,
	chunkStart int,
	logger *slog.Logger,
	processingErrors *[]error,
) []members.Member {
	chunkMembers := make([]members.Member, 0, len(chunk))

	for i, discordMember := range chunk {
		select {
		case <-ctx.Done():
			logger.Info("member chunk processing cancelled")
			return chunkMembers
		default:
		}

		time.Sleep(1 * time.Second)

		globalIndex := chunkStart + i
		mlogger := logger.With(
			"id", discordMember.User.ID,
			"name", discordMember.DisplayName(),
			"index", globalIndex)

		mlogger.Debug("updating member")

		// Get or create member
		member, err := getOrCreateMember(discordMember, mlogger)
		if err != nil {
			mlogger.Error("getting or creating member", "error", err)
			*processingErrors = append(*processingErrors, err)
			continue
		}

		if member == nil {
			continue
		}

		if err := member.UpdateFromDiscordMember(discordMember); err != nil {
			*processingErrors = append(*processingErrors, errors.Wrap(err, "updating member from discord member"))
			continue
		}

		if err := member.UpdateRsiInfo(); err != nil {
			*processingErrors = append(*processingErrors, errors.Wrap(err, "updating RSI info"))
			continue
		}

		mlogger.Debug("adding member to chunk save batch")
		chunkMembers = append(chunkMembers, *member)
	}

	return chunkMembers
}

// getOrCreateMember retrieves an existing member or creates a new one
func getOrCreateMember(discordMember *discordgo.Member, logger *slog.Logger) (*members.Member, error) {
	logger.Debug("getting stored member")
	member, err := members.Get(discordMember.User.ID)
	if err != nil {
		if !errors.Is(err, members.MemberNotFound) {
			return nil, err
		}

		logger.Debug("creating new member")
		member = members.New(discordMember)
	}

	logger.Debug("member retrieved/created")
	return member, nil
}
