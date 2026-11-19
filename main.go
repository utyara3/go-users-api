package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := newLogger()

	if err := run(logger); err != nil {
		logger.Error("application terminated", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := context.Background()
	pool, err := NewDB(ctx)
	if err != nil {
		return fmt.Errorf("initialize database pool: %w", err)
	}
	defer pool.Close()

	usersRepo := &PostgresUsers{db: pool}
	usersService := &UserService{userRepo: usersRepo}

	srv := &Server{userService: usersService}

	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           newMux(srv),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErr := make(chan error, 1)

	go func() {
		logger.Info("starting HTTP server", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("Server error: %w", err)
		}
	case <-shutdownCtx.Done():
		logger.Info("Shutdown signal received")
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(cancelCtx); shutdownErr != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("graceful shutdown %w", shutdownErr),
				fmt.Errorf("force close: %w", closeErr),
			)
		}
		return fmt.Errorf("graceful shutdown: %w", shutdownErr)
	}

	logger.Info("Server stopped")

	return nil
}
