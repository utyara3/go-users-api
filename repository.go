package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUsers struct {
	db *pgxpool.Pool
}

type UserRepository interface {
	Get(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, name string) (*User, error)
	Update(ctx context.Context, id int64, name string) (*User, error)
	Delete(ctx context.Context, id int64) error
}

var (
	ErrUserNotFound = errors.New("user not found")

	// In compile stage check that PostgresUsers realise UserRepository interface
	_ UserRepository = (*PostgresUsers)(nil)
)

func (p *PostgresUsers) Get(ctx context.Context, id int64) (*User, error) {
	var user User

	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("getting user: %w", err)
	}

	return &user, nil
}

func (p *PostgresUsers) Create(ctx context.Context, name string) (*User, error) {
	var user User

	err := p.db.QueryRow(ctx, "INSERT INTO users (name) VALUES ($1) RETURNING id, name", name).Scan(&user.ID, &user.Name)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return &user, nil
}

func (p *PostgresUsers) Update(ctx context.Context, id int64, name string) (*User, error) {
	var updatedUser User

	query := `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id, name
	`
	err := p.db.QueryRow(ctx, query, name, id).Scan(&updatedUser.ID, &updatedUser.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
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
		return ErrUserNotFound
	}

	return nil
}
