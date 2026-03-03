package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/settings"
)

func MonitorAttendance(ctx context.Context, logger *slog.Logger, stop <-chan struct{}) {
	logger = logger.With("func", "bot.MonitorAttendance")
	logger.Info("monitoring attendance")

	attendanceChannelId := settings.GetString("ATTENDANCE_CHANNEL_ID")
	if attendanceChannelId == "" {
		logger.Error("attendance channel id not set in settings")
		return
	}

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			logger.Warn("stopping monitor")
			return
		case <-ticker.C:
		}

		if needsSetup() {
			logger.Warn("initial setup not complete, skipping attendance check")
			continue
		}

		attendanceMessages := []*discordgo.Message{}
		latestId := ""
	AGAIN:
		logger.Debug("checking attendance messages")
		msgs, err := bot.ChannelMessages(attendanceChannelId, 100, latestId, "", "")
		if err != nil {
			logger.Error("failed to get messages", "error", err)
			continue
		}

		logger.Debug(fmt.Sprintf("got %d messages", len(msgs)))

		if len(attendanceMessages) != 0 {
			for _, msg := range msgs {
				if msg.Author.ID == bot.ClientId {
					attendanceMessages = append(attendanceMessages, msg)
					latestId = msg.ID
				}
			}
			goto AGAIN
		}

		for _, msg := range msgs {
			id := msg.Embeds[0].Description
			_, err := attdnc.Get(id)
			if (err != nil && errors.Is(err, attdnc.ErrAttendanceNotFound)) || id == "" {
				_ = bot.ChannelMessageDelete(attendanceChannelId, msg.ID)
			}
		}

		<-ticker.C
	}
}

// needsSetup checks if initial setup is required
func needsSetup() bool {
	// Check if setup was completed
	if settings.GetBoolWithDefault("setup_completed", false) {
		return false
	}

	// Check critical Discord settings
	criticalSettings := []string{
		"RSI_ORG_ID",
		"RECRUIT_ROLE_ID",
		"ALLY_ROLE_ID",
		"MEMBER_ROLE_ID",
		"TECHNICIAN_ROLE_ID",
		"SPECIALIST_ROLE_ID",
		"LIEUTENANT_ROLE_ID",
		"COMMANDER_ROLE_ID",
		"ADMIRAL_ROLE_ID",
		"RSI_TOKEN",
		"RSI_DEVICE",
	}

	for _, setting := range criticalSettings {
		value := os.Getenv(setting)
		if value == "" {
			return true // Missing critical setting
		}
	}

	return false
}
