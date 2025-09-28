package utils

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

var testSkipError = errors.New("skip retry error")
var testRetryError = errors.New("retry error")

func TestExponentialBackoff_SkipRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with skip condition that skips testSkipError
	backoff := NewExponentialBackoffWithSkipCondition(
		10*time.Millisecond,
		100*time.Millisecond,
		2.0,
		3, // max retries
		logger,
		func(err error) bool {
			return errors.Is(err, testSkipError)
		},
	)

	attempts := 0
	err := backoff.Execute(func() error {
		attempts++
		return testSkipError
	})

	// Should not retry when skip condition is met
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
	if !errors.Is(err, testSkipError) {
		t.Errorf("expected testSkipError, got %v", err)
	}
}

func TestExponentialBackoff_NoSkipRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with skip condition but error that should retry
	backoff := NewExponentialBackoffWithSkipCondition(
		10*time.Millisecond,
		100*time.Millisecond,
		2.0,
		2, // max retries
		logger,
		func(err error) bool {
			return errors.Is(err, testSkipError)
		},
	)

	attempts := 0
	err := backoff.Execute(func() error {
		attempts++
		return testRetryError
	})

	// Should retry when skip condition is not met (1 initial + 2 retries = 3 total)
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, testRetryError) {
		t.Errorf("expected testRetryError, got %v", err)
	}
}

func TestExponentialBackoff_NoSkipCondition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test original constructor without skip condition
	backoff := NewExponentialBackoff(
		10*time.Millisecond,
		100*time.Millisecond,
		2.0,
		2, // max retries
		logger,
	)

	attempts := 0
	err := backoff.Execute(func() error {
		attempts++
		return testSkipError
	})

	// Should retry when no skip condition is set (1 initial + 2 retries = 3 total)
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, testSkipError) {
		t.Errorf("expected testSkipError, got %v", err)
	}
}
