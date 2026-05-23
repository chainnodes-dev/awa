// Package logger initialises the global slog logger.
//
// Call Init early in main() before any other package logs. After Init,
// all slog.Info/Warn/Error/Debug calls produce structured JSON output.
// The log level is read from config (LOG_LEVEL env var) and defaults to info.
//
// Request IDs are propagated via context using WithRequestID / RequestIDFromContext.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey struct{}

// Init configures the global slog logger.
// logLevel is one of "debug", "info", "warn", "error" (case-insensitive).
// Output is JSON on stdout — structured and machine-parseable.
func Init(logLevel string) {
	level := parseLevel(logLevel)
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}

// WithRequestID returns a child context carrying the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKey{}, requestID)
}

// RequestIDFromContext extracts the request ID from ctx, or returns "".
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKey{}).(string); ok {
		return v
	}
	return ""
}

// FromContext returns an slog.Logger with the request_id attribute pre-attached
// when one is present in ctx. Use this inside handlers and engine methods.
func FromContext(ctx context.Context) *slog.Logger {
	if rid := RequestIDFromContext(ctx); rid != "" {
		return slog.With("request_id", rid)
	}
	return slog.Default()
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
