package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PostgresUsers struct {
	db *pgxpool.Pool
}

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (User, error)
	Create(ctx context.Context, name string) (int64, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, id int64) error
}

var (
	ErrUserNotFound error = errors.New("user not found")

	// In compile stage check that PostgresUsers realise UserRepository interface
	_ UserRepository = (*PostgresUsers)(nil)
)

func (p *PostgresUsers) GetByID(ctx context.Context, id int64) (User, error) {
	var user User

	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}

		return User{}, fmt.Errorf("getting user: %w", err)
	}

	return user, nil
}

func (p *PostgresUsers) Create(ctx context.Context, name string) (int64, error) {
	var lastUserID int64

	err := p.db.QueryRow(ctx, "INSERT INTO users (name) VALUES ($1) RETURNING id", name).Scan(&lastUserID)
	if err != nil {
		return 0, fmt.Errorf("creating user: %w", err)
	}

	return lastUserID, err
}

func (p *PostgresUsers) Update(ctx context.Context, user User) (User, error) {
	var updatedUser User

	query := `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id, name
	`
	err := p.db.QueryRow(ctx, query, user.Name, user.ID).Scan(&updatedUser.ID, &updatedUser.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("updating user: %w", err)
	}

	return updatedUser, nil
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
