package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

type Server struct {
	users UserGetter
}

type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (User, error)
}

func (p *PostgresUsers) GetUserByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := p.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Server) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

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
