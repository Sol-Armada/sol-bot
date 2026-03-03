package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/settings"
)

func MonitorAttendance(ctx context.Context, logger *slog.Logger, stop <-chan struct{}) {
	logger = logger.With("func", "bot.MonitorAttendance")
	logger.Info("monitoring attendance")

	attendanceChannelId := settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID")
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
