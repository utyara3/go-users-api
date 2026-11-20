package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utyara3/go-users-api/internal/postgres"
	"github.com/utyara3/go-users-api/internal/user"
)

/*
Integration tests for the real Postgres repository.
*/
func newIntegrationRepo(t *testing.T) (*postgres.PostgresUsers, *pgxpool.Pool) {
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

	return postgres.NewUserRepository(pool), pool
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
	created, err := repo.Create(ctx, name)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	id := created.ID
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	t.Cleanup(func() { cleanupUser(t, pool, id) })

	// Get
	getUser, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if getUser.ID != id || getUser.Name != name {
		t.Fatalf("expected user %d %q, got %d %q", id, name, getUser.ID, getUser.Name)
	}

	// Update
	updatedName := name + "-updated"
	updated, err := repo.Update(ctx, id, updatedName)
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updated.ID != id || updated.Name != updatedName {
		t.Fatalf("expected user %d %q, got %d %q", id, updatedName, updated.ID, updated.Name)
	}

	getUser, err = repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if getUser.Name != updatedName {
		t.Fatalf("expected updated name %q, got %q", updatedName, getUser.Name)
	}

	// Delete
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Operations on a deleted user must return ErrUserNotFound
	if _, err := repo.Get(ctx, id); !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound after delete, got: %v", err)
	}

	if err := repo.Delete(ctx, id); !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound on second delete, got: %v", err)
	}

	if _, err := repo.Update(ctx, id, "ghost"); !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound on update, got: %v", err)
	}
}

func TestIntegration_PostgresUsersGetNotFound(t *testing.T) {
	repo, _ := newIntegrationRepo(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := repo.Get(ctx, math.MaxInt64)
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}
