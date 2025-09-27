package utils

import (
	"log/slog"
	"math"
	"time"
)

type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	MaxRetries   int
	logger       *slog.Logger
}

func NewExponentialBackoff(initialDelay, maxDelay time.Duration, multiplier float64, maxRetries int, logger *slog.Logger) *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		MaxRetries:   maxRetries,
		logger:       logger,
	}
}

func (eb *ExponentialBackoff) Execute(operation func() error) error {
	var err error
	delay := eb.InitialDelay

	for attempt := 0; attempt <= eb.MaxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		if attempt == eb.MaxRetries {
			break
		}

		eb.logger.Warn("operation failed, retrying with backoff",
			"attempt", attempt+1,
			"max_retries", eb.MaxRetries,
			"delay", delay,
			"error", err)

		time.Sleep(delay)

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * eb.Multiplier)
		if delay > eb.MaxDelay {
			delay = eb.MaxDelay
		}
	}

	return err
}

func (eb *ExponentialBackoff) CalculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(eb.InitialDelay) * math.Pow(eb.Multiplier, float64(attempt)))
	if delay > eb.MaxDelay {
		delay = eb.MaxDelay
	}
	return delay
}
