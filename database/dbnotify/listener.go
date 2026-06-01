package dbnotify

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sol-armada/sol-bot/database"
)

var channelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Handler consumes decoded events from subscribed LISTEN channels.
type Handler func(ctx context.Context, event Event) error

// ListenerConfig controls connection and reconnect behavior.
type ListenerConfig struct {
	DSN string

	Channels []string

	ConnectTimeout    time.Duration
	MinReconnectDelay time.Duration
	MaxReconnectDelay time.Duration

	OnError func(error)
}

// Listener keeps a dedicated connection open for LISTEN/NOTIFY.
type Listener struct {
	cfg ListenerConfig
}

// NewListener constructs a reconnecting LISTEN/NOTIFY consumer.
func NewListener(cfg ListenerConfig) (*Listener, error) {
	if cfg.DSN == "" {
		return nil, errors.New("listener dsn is required")
	}
	if len(cfg.Channels) == 0 {
		return nil, errors.New("at least one channel is required")
	}

	for _, channel := range cfg.Channels {
		if !channelNamePattern.MatchString(channel) {
			return nil, fmt.Errorf("invalid channel name %q", channel)
		}
	}

	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.MinReconnectDelay <= 0 {
		cfg.MinReconnectDelay = 500 * time.Millisecond
	}
	if cfg.MaxReconnectDelay <= 0 {
		cfg.MaxReconnectDelay = 30 * time.Second
	}
	if cfg.MaxReconnectDelay < cfg.MinReconnectDelay {
		cfg.MaxReconnectDelay = cfg.MinReconnectDelay
	}

	return &Listener{cfg: cfg}, nil
}

// NewListenerFromPostgresConfig builds a listener from database.PostgresConfig.
func NewListenerFromPostgresConfig(cfg database.PostgresConfig, channels []string) (*Listener, error) {
	return NewListener(ListenerConfig{
		DSN:      cfg.DSN(),
		Channels: channels,
	})
}

// Run blocks and processes notifications until context cancellation.
func (l *Listener) Run(ctx context.Context, handler Handler) error {
	if handler == nil {
		return errors.New("listener handler is required")
	}

	delay := l.cfg.MinReconnectDelay

	for {
		if ctx.Err() != nil {
			return nil
		}

		conn, err := l.connect(ctx)
		if err != nil {
			l.reportError(fmt.Errorf("connect listener: %w", err))
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil
			}
			delay = nextDelay(delay, l.cfg.MaxReconnectDelay)
			continue
		}

		if err := l.listenChannels(ctx, conn); err != nil {
			l.reportError(fmt.Errorf("subscribe channels: %w", err))
			_ = conn.Close(ctx)
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil
			}
			delay = nextDelay(delay, l.cfg.MaxReconnectDelay)
			continue
		}

		delay = l.cfg.MinReconnectDelay
		err = l.consume(ctx, conn, handler)
		_ = conn.Close(ctx)

		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}

		l.reportError(err)
		if err := sleepWithContext(ctx, delay); err != nil {
			return nil
		}
		delay = nextDelay(delay, l.cfg.MaxReconnectDelay)
	}
}

func (l *Listener) connect(ctx context.Context) (*pgx.Conn, error) {
	connectCtx, cancel := context.WithTimeout(ctx, l.cfg.ConnectTimeout)
	defer cancel()
	return pgx.Connect(connectCtx, l.cfg.DSN)
}

func (l *Listener) listenChannels(ctx context.Context, conn *pgx.Conn) error {
	for _, channel := range l.cfg.Channels {
		query := "LISTEN " + channel
		if _, err := conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("listen %s: %w", channel, err)
		}
	}
	return nil
}

func (l *Listener) consume(ctx context.Context, conn *pgx.Conn, handler Handler) error {
	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return wrapWaitError(err)
		}

		event, err := ParseEvent(notification.Channel, notification.Payload)
		if err != nil {
			l.reportError(fmt.Errorf("parse %s payload: %w", notification.Channel, err))
			continue
		}

		if err := handler(ctx, event); err != nil {
			l.reportError(fmt.Errorf("handle %s notification: %w", notification.Channel, err))
		}
	}
}

func (l *Listener) reportError(err error) {
	if err == nil {
		return
	}
	if l.cfg.OnError != nil {
		l.cfg.OnError(err)
	}
}

func nextDelay(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func wrapWaitError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("wait notification pg error: %w", err)
	}

	return fmt.Errorf("wait notification: %w", err)
}
