package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PostgresUsers struct {
	db *pgxpool.Pool
}

type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (User, error)
}

type UserCreator interface {
	CreateNewUser(ctx context.Context, name string) (int64, error)
}

// In compile stage check that PostgresUsers realise UserGetter interface
var (
	_ UserGetter  = (*PostgresUsers)(nil)
	_ UserCreator = (*PostgresUsers)(nil)
)

func (p *PostgresUsers) GetUserByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (p *PostgresUsers) CreateNewUser(ctx context.Context, name string) (int64, error) {
	var lastUserID int64

	err := p.db.QueryRow(ctx, "INSERT INTO users (name) VALUES ($1) RETURNING id", name).Scan(&lastUserID)
	if err != nil {
		return 0, err
	}

	return lastUserID, err
}
