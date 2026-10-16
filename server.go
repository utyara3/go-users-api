package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type Server struct {
	users UserGetter
}

type JSONError struct {
	Err string `json:"error"`
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(JSONError{Err: message}); err != nil {
		log.Printf("encoding response error: %v", err)
	}
}

func (s *Server) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || userID <= 0 {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			sendJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		sendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("encoding user response error: %v", err)
	}
}
