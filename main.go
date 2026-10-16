package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()

	pool, err := NewDB(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}

	defer pool.Close()

	fmt.Println("Connected to PostgreSQL")

	port := ":8080"
	users := &PostgresUsers{db: pool}
	srv := &Server{users: users}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{id}", srv.GetUserByIDHandler)

	log.Printf("Сервер запущен на %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Printf("Server error: %v", err)
		return
	}
}
