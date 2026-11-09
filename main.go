package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	pool, err := NewDB(ctx)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	defer pool.Close()

	usersRepo := &PostgresUsers{db: pool}
	srv := &Server{usersRepo: usersRepo}

	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           newMux(srv),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErr := make(chan error, 1)

	go func() {
		log.Println("Starting server on port 8080")
		serverErr <- server.ListenAndServe()
	}()

	shutdownCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("Server error: %w", err)
		}
	case <-shutdownCtx.Done():
		log.Println("Shutdown signal received")
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(cancelCtx); err != nil {
		log.Printf("Server shutdown error: %v\n", err)

		if err := server.Close(); err != nil {
			log.Printf("Server close error: %v\n", err)
		}
	}

	log.Println("Server stopped")

	return nil
}
