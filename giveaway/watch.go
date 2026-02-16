package giveaway

import (
	"log/slog"
	"time"
)

func watch() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		if len(giveaways) == 0 {
			continue
		}

		for _, g := range giveaways {
			if g.Ended {
				slog.Debug("giveaway timer ended")
				delete(giveaways, g.Id)
				continue
			}

			if time.Now().After(g.EndTime) {
				slog.Debug("giveaway timer expired")
				if err := g.End(); err != nil {
					slog.Error("failed to end giveaway", "error", err)
				}
				delete(giveaways, g.Id)
				continue
			}

			r := max(time.Until(g.EndTime), 0)
			hours := int(r.Hours())
			minutes := int(r.Minutes()) % 60
			seconds := int(r.Seconds()) % 60

			switch {
			case hours > 6 && minutes == 0 && seconds == 0:
				slog.Debug("Performing hourly action", "remaining_hours", hours)
			case hours <= 6 && minutes%30 == 0 && seconds == 0:
				slog.Debug("Performing 30-minute action", "remaining_minutes", int(r.Minutes()))
			case hours == 0 && minutes > 10 && minutes%10 == 0 && seconds == 0:
				slog.Debug("Performing 10-minute action", "remaining_minutes", int(r.Minutes()))
			case hours == 0 && minutes <= 10 && seconds == 0:
				slog.Debug("Performing 1-minute action", "remaining_seconds", int(r.Seconds()))
			case hours == 0 && minutes < 2:
				slog.Debug("Performing 1-second action", "remaining_seconds", int(r.Seconds()))
			default:
				continue
			}

			if err := g.UpdateMessage(); err != nil {
				slog.Error("failed to update giveaway message", "error", err)
			}
		}
	}
}
