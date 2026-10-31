package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewDBWithoutConnectionURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	pool, err := NewDB(context.Background())
	if pool != nil {
		t.Fatal("expected nil pool when DATABASE_URL is not set")
	}

	if !errors.Is(err, ErrDBNotFound) {
		t.Fatalf("expected ErrDBNotFound, got: %v", err)
	}
}

func TestNewDBPingFailure(t *testing.T) {
	// Port 1 is not usable, so ping fails fast without any external services.
	t.Setenv("DATABASE_URL", "postgres://app:secret@127.0.0.1:1/app?connect_timeout=1")

	pool, err := NewDB(context.Background())
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
