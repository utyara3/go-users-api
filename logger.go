package main

import (
	"fmt"
	"log/slog"
	"os"
)

func newLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevelFromEnv(),
	})

	return slog.New(handler)
}

// logLevelFromEnv reads LOG_LEVEL (debug, info, warn, error).
// Missing or invalid values fall back to info.
func logLevelFromEnv() slog.Level {
	raw := os.Getenv("LOG_LEVEL")
	if raw == "" {
		return slog.LevelInfo
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid LOG_LEVEL %q, falling back to info: %v\n", raw, err)
		return slog.LevelInfo
	}

	return level
}
