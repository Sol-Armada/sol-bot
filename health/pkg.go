package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/sol-armada/sol-bot/stores"
)

var healthy bool = false

func Monitor() {
	logger := slog.Default().With("func", "health.Monitor")
	for {
		s := stores.Get()
		ctx := context.Background()
		if !s.Connected(ctx) {
			logger.Warn("not connected to storage")
			healthy = false
			goto WAIT
		}

		healthy = true
	WAIT:
		time.Sleep(10 * time.Second)
	}
}

func IsHealthy() bool {
	return healthy
}
