package main

import (
	"context"
	"log/slog"
	"testing"
)

func TestLogLevelFromEnv(t *testing.T) {
	testCases := []struct {
		Name      string
		Level     string
		WantDebug bool
		WantInfo  bool
	}{
		{Name: "debug level", Level: "debug", WantDebug: true, WantInfo: true},
		{Name: "info level", Level: "info", WantDebug: false, WantInfo: true},
		{Name: "error level", Level: "error", WantDebug: false, WantInfo: false},
		{Name: "invalid value falls back to info", Level: "banana", WantDebug: false, WantInfo: true},
		{Name: "empty value falls back to info", Level: "", WantDebug: false, WantInfo: true},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tc.Level)

			logger := newLogger()
			ctx := context.Background()

			if got := logger.Enabled(ctx, slog.LevelDebug); got != tc.WantDebug {
				t.Fatalf("debug enabled = %v, want %v", got, tc.WantDebug)
			}

			if got := logger.Enabled(ctx, slog.LevelInfo); got != tc.WantInfo {
				t.Fatalf("info enabled = %v, want %v", got, tc.WantInfo)
			}
		})
	}
}
