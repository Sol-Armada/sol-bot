package utils

import (
	"context"
	"log/slog"
)

type ContextKey string

const (
	MEMBER ContextKey = "member"
	LOGGER ContextKey = "logger"
)

func SetMemberToContext(ctx context.Context, member any) context.Context {
	return context.WithValue(ctx, MEMBER, member)
}

func GetMemberFromContext(ctx context.Context) any {
	return ctx.Value(MEMBER)
}

func SetLoggerToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LOGGER, logger)
}

func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if logger := ctx.Value(LOGGER); logger != nil {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}
