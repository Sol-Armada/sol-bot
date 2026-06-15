package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/sol-armada/sol-bot/database"
)

var healthy bool = false

func Monitor() {
	logger := slog.Default().With("func", "health.Monitor")
	for {
		connected := false
		s := database.Get()
		if s == nil || s.Pool == nil {
			logger.Warn("database client not initialized")
		} else {
			ctx := context.Background()
			if err := s.Pool.Ping(ctx); err != nil {
				logger.Warn("not connected to storage")
			} else {
				connected = true
			}
		}

		healthy = connected
		time.Sleep(10 * time.Second)
	}
}

func IsHealthy() bool {
	return healthy
}
