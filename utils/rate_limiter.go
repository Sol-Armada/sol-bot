package utils

import (
	"log/slog"
	"time"
)

type ExponentialBackoff struct {
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	MaxRetries      int
	logger          *slog.Logger
	shouldSkipRetry func(error) bool
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

// NewExponentialBackoffWithSkipCondition creates a new ExponentialBackoff with a custom skip condition
func NewExponentialBackoffWithSkipCondition(initialDelay, maxDelay time.Duration, multiplier float64, maxRetries int, logger *slog.Logger, shouldSkipRetry func(error) bool) *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay:    initialDelay,
		MaxDelay:        maxDelay,
		Multiplier:      multiplier,
		MaxRetries:      maxRetries,
		logger:          logger,
		shouldSkipRetry: shouldSkipRetry,
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

		// Check if we should skip retries for this error
		if eb.shouldSkipRetry != nil && eb.shouldSkipRetry(err) {
			eb.logger.Debug("skipping retry due to error condition",
				"error", err,
				"attempt", attempt+1)
			return err
		}

		if attempt == eb.MaxRetries {
			break
		}

		eb.logger.Warn("operation failed, retrying with backoff",
			"attempt", attempt+1,
			"max_retries", eb.MaxRetries,
			"delay_seconds", delay.Seconds(),
			"error", err)

		time.Sleep(delay)

		// Calculate next delay with exponential backoff
		delay = min(time.Duration(float64(delay)*eb.Multiplier), eb.MaxDelay)
	}

	return err
}
