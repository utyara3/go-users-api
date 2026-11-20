package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/utyara3/go-users-api/internal/postgres"
)

func TestNewDBWithoutConnectionURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	pool, err := postgres.NewDB(context.Background())
	if pool != nil {
		t.Fatal("expected nil pool when DATABASE_URL is not set")
	}

	if !errors.Is(err, postgres.ErrDBNotFound) {
		t.Fatalf("expected ErrDBNotFound, got: %v", err)
	}
}

func TestNewDBPingFailure(t *testing.T) {
	// Port 1 is not usable, so ping fails fast without any external services.
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?connect_timeout=1")

	pool, err := postgres.NewDB(context.Background())
	if pool != nil {
		t.Fatal("expected nil pool when postgres is unreachable")
	}

	if err == nil {
		t.Fatal("expected an error when postgres is unreachable")
	}

	if !strings.Contains(err.Error(), "ping postgres") {
		t.Fatalf("expected error to be wrapped with 'ping postgres', got: %v", err)
	}
}
