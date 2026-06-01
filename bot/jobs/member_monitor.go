package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

const (
	memberProcessingChunkSize = 10
	discordAPIRetries         = 3

	discordAPIMaxDelay = 5 * time.Minute
	healthCheckDelay   = 10 * time.Second
)

// MemberMonitor is the main function that orchestrates the member monitoring process.
// It fetches Discord members, validates them, updates their information in the database,
// and cleans up members who are no longer in the Discord server.
func MemberMonitor(ctx context.Context, s *discordgo.Session, monitor JobMonitor) error {
	logger := slog.Default()
	defer func() {
		if err := recover(); err != nil {
			logger.Error("panic in member monitor", "error", err)
		}
	}()

	logger.Debug("MemberMonitor starting")

	if !health.IsHealthy() {
		time.Sleep(healthCheckDelay)
		return errors.New("not healthy")
	}

	start := time.Now().UTC()
	logger.Info("scanning members")

	validDiscordMembers, err := fetchAndValidateDiscordMembers(s, logger)
	if err != nil {
		return errors.Wrap(err, "failed to fetch discord members")
	}

	logger.Info(fmt.Sprintf("updating %d members", len(validDiscordMembers)))
	if err := updateMembers(ctx, logger, monitor, validDiscordMembers); err != nil {
		return errors.Wrap(err, "updating members")
	}

	monitor.Update("Cleaning up removed members")
	if err := cleanupRemovedMembers(ctx, logger, validDiscordMembers); err != nil {
		return errors.Wrap(err, "cleaning up removed members")
	}

	monitor.Update(fmt.Sprintf("Member monitor complete (%d members)", len(validDiscordMembers)))
	logger.Info("members updated", "count", len(validDiscordMembers), "duration", time.Since(start))
	return nil
}

func updateMembers(ctx context.Context, logger *slog.Logger, monitor JobMonitor, discordMembers []*discordgo.Member) error {
	logger.Debug("checking members", "discord_members", len(discordMembers))

	if len(discordMembers) == 0 {
		logger.Info("no members to process")
		return nil
	}

	var processingErrors []error

	for chunkStart := 0; chunkStart < len(discordMembers); chunkStart += memberProcessingChunkSize {
		select {
		case <-ctx.Done():
			logger.Info("member update cancelled")
			return nil
		default:
		}

		chunkEnd := min(chunkStart+memberProcessingChunkSize, len(discordMembers))

		chunk := discordMembers[chunkStart:chunkEnd]

		logger.Info("processing chunk", "start", chunkStart, "end", chunkEnd, "size", len(chunk))
		monitor.Update(fmt.Sprintf("Updating members... (%d/%d)", chunkEnd, len(discordMembers)))

		// Process each member in the chunk
		processedMembers := processChunkMembers(ctx, chunk, chunkStart, logger, &processingErrors)

		// Save chunk in batch
		if len(processedMembers) > 0 {
			logger.Debug("saving chunk", "count", len(processedMembers))
			if err := members.BulkSave(processedMembers); err != nil {
				processingErrors = append(processingErrors, err)
				continue
			}
			logger.Debug("chunk saved successfully", "count", len(processedMembers))
		}
	}

	if len(processingErrors) > 0 {
		for _, err := range processingErrors {
			logger.Error("error processing member", "error", err)
		}
		return errors.New("one or more errors occurred while processing members")
	}

	return nil
}

// fetchAndValidateDiscordMembers fetches Discord members with retry logic and validates them
func fetchAndValidateDiscordMembers(s *discordgo.Session, logger *slog.Logger) ([]*discordgo.Member, error) {
	backoff := utils.NewExponentialBackoff(
		1*time.Second,      // initial delay
		discordAPIMaxDelay, // max delay
		2.0,                // multiplier
		discordAPIRetries,  // max retries
		logger,
	)

	var discordMembers []*discordgo.Member
	if err := backoff.Execute(func() error {
		if s.Ratelimiter != nil {
			rateBucket := s.Ratelimiter.GetBucket("guild_member_check")
			if rateBucket != nil && rateBucket.Remaining == 0 {
				logger.Warn("hit a rate limit. waiting for rate limit to reset")
				time.Sleep(s.Ratelimiter.GetWaitTime(rateBucket, 0))
			}
		}
		var err error
		discordMembers, err = s.GuildMembers(settings.GetString("discord.guild_id"), "", 1000)
		if err != nil {
			return errors.Wrap(err, "fetching members")
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to get discord members after retries")
	}

	logger.Info("discord members fetched", "count", len(discordMembers))

	validMembers := members.ValidateDiscordMembers(discordMembers)
	logger.Info("discord members validated", "valid_count", len(validMembers), "filtered_out", len(discordMembers)-len(validMembers))

	return validMembers, nil
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
			deleted, err := deleteStoredMemberIfNeeded(logger, storedMemberID)
			if err != nil {
				logger.Error("error deleting stored member", "member_id", storedMemberID, "error", err)
				continue
			}
			if deleted {
				deletedCount++
			}
		}
	}

	logger.Info("cleanup completed", "deleted_count", deletedCount)
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
func deleteStoredMemberIfNeeded(logger *slog.Logger, storedMemberID string) (bool, error) {
	memberLogger := logger.With("stored_member_id", storedMemberID)
	memberLogger.Debug("deleting stored member", "reason", "not_in_discord")

	storedMember, getMemberErr := members.Get(storedMemberID)
	if getMemberErr != nil && !errors.Is(getMemberErr, members.MemberNotFound) {
		return false, errors.Wrap(getMemberErr, "getting member for deletion check")
	}

	if errors.Is(getMemberErr, members.MemberNotFound) {
		return false, nil
	}

	if storedMember.IsBot {
		memberLogger.Debug("skipping bot member deletion")
		return false, nil
	}

	if err := storedMember.Delete("Bot did not find on server"); err != nil {
		return false, errors.Wrap(err, "deleting member")
	}

	memberLogger.Debug("member deleted successfully")
	return true, nil
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
			*processingErrors = append(*processingErrors, errors.Wrap(err, "getting member"))
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

		chunkMembers = append(chunkMembers, *member)
	}

	return chunkMembers
}

// getOrCreateMember retrieves an existing member or creates a new one
func getOrCreateMember(discordMember *discordgo.Member, logger *slog.Logger) (*members.Member, error) {
	member, err := members.Get(discordMember.User.ID)
	if err != nil {
		if !errors.Is(err, members.MemberNotFound) {
			return nil, err
		}

		logger.Debug("creating new member")
		member = members.New(discordMember)
	}

	logger.Debug("got stored member")
	return member, nil
}
