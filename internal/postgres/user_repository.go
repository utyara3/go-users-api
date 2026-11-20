package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utyara3/go-users-api/internal/user"
)

type PostgresUsers struct {
	db *pgxpool.Pool
}

// In compile stage check that PostgresUsers realise UserRepository interface
var _ user.UserRepository = (*PostgresUsers)(nil)

func NewUserRepository(db *pgxpool.Pool) *PostgresUsers {
	return &PostgresUsers{db: db}
}

func (p *PostgresUsers) Get(ctx context.Context, id int64) (*user.User, error) {
	var dbUser user.User

	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&dbUser.ID, &dbUser.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("getting user: %w", err)
	}

	return &dbUser, nil
}

func (p *PostgresUsers) Create(ctx context.Context, name string) (*user.User, error) {
	var dbUser user.User

	err := p.db.QueryRow(ctx, "INSERT INTO users (name) VALUES ($1) RETURNING id, name", name).Scan(&dbUser.ID, &dbUser.Name)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return &dbUser, nil
}

func (p *PostgresUsers) Update(ctx context.Context, id int64, name string) (*user.User, error) {
	var updatedUser user.User

	query := `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id, name
	`
	err := p.db.QueryRow(ctx, query, name, id).Scan(&updatedUser.ID, &updatedUser.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &updatedUser, nil
}

func (p *PostgresUsers) Delete(ctx context.Context, id int64) error {
	tag, err := p.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}
