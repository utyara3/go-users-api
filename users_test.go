package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

/*
Integration tests for the real Postgres repository.
*/
func newIntegrationRepo(t *testing.T) (*PostgresUsers, *pgxpool.Pool) {
	t.Helper()

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id   BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`
	if _, err := pool.Exec(ctx, schema); err != nil {
		t.Fatalf("create users table: %v", err)
	}

	return &PostgresUsers{db: pool}, pool
}

func cleanupUser(t *testing.T, pool *pgxpool.Pool, id int64) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id); err != nil {
		t.Logf("cleanup: deleting user %d: %v", id, err)
	}
}

func TestIntegration_PostgresUsersCRUD(t *testing.T) {
	repo, pool := newIntegrationRepo(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("integration-user-%d", time.Now().UnixNano())

	// Create
	id, err := repo.Create(ctx, name)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	t.Cleanup(func() { cleanupUser(t, pool, id) })

	// GetByID
	user, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.ID != id || user.Name != name {
		t.Fatalf("expected user %d %q, got %d %q", id, name, user.ID, user.Name)
	}

	// Update
	updatedName := name + "-updated"
	updated, err := repo.Update(ctx, User{ID: id, Name: updatedName})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updated.ID != id || updated.Name != updatedName {
		t.Fatalf("expected user %d %q, got %d %q", id, updatedName, updated.ID, updated.Name)
	}

	user, err = repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if user.Name != updatedName {
		t.Fatalf("expected updated name %q, got %q", updatedName, user.Name)
	}

	// Delete
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Operations on a deleted user must return ErrUserNotFound
	if _, err := repo.GetByID(ctx, id); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound after delete, got: %v", err)
	}

	if err := repo.Delete(ctx, id); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound on second delete, got: %v", err)
	}

	if _, err := repo.Update(ctx, User{ID: id, Name: "ghost"}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound on update, got: %v", err)
	}
}

func TestIntegration_PostgresUsersGetNotFound(t *testing.T) {
	repo, _ := newIntegrationRepo(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := repo.GetByID(ctx, math.MaxInt64)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}
