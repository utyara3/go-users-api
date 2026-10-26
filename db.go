package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDBNotFound error = errors.New("database not found")

func NewDB(ctx context.Context) (*pgxpool.Pool, error) {
	/*
		postgres:// - connection protocol
		app:secret - user && password
		localhost - postgresql adders relative to app
		5432 - port
		/app - db name
	*/
	// connString := "postgres://app:secret@localhost:5432/app"
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return nil, fmt.Errorf("%w: %v", ErrDBNotFound, connString)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
