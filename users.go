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

// In compile stage check that PostgresUsers realise UserGetter interface
var _ UserGetter = (*PostgresUsers)(nil)

func (p *PostgresUsers) GetUserByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
