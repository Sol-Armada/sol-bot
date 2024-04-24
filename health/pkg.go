package health

import (
	"time"

	"github.com/apex/log"
	"github.com/sol-armada/sol-bot/stores"
)

var healthy bool = false

func Monitor() {
	logger := log.WithField("func", "health.Monitor")
	for {
		if !stores.Connected() {
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
