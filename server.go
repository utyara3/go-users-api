package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	usersRepo UserRepository
}

type JSONError struct {
	Err string `json:"error"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
}

type ChangeUserNameRequest struct {
	Name string `json:"name"`
}

func newMux(srv *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", srv.GetUserByIDHandler)
	mux.HandleFunc("POST /users", srv.CreateNewUserHandler)
	mux.HandleFunc("PUT /users/{id}", srv.ChangeUserNameHandler)
	mux.HandleFunc("DELETE /users/{id}", srv.DeleteUserHandler)
	return mux
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(JSONError{Err: message}); err != nil {
		log.Printf("encoding response error: %v", err)
	}
}

func (srv *Server) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || userID <= 0 {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	user, err := srv.usersRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
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

func (srv *Server) CreateNewUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		sendJSONError(w, "Name is empty", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	name := strings.TrimSpace(req.Name)
	userID, err := srv.usersRepo.Create(ctx, name)
	if err != nil {
		sendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	user := User{ID: userID, Name: name}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("encoding user response error: %v", err)
	}
}

func (srv *Server) ChangeUserNameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req ChangeUserNameRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		sendJSONError(w, "Name is empty", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	updatedUser, err := srv.usersRepo.Update(ctx, User{ID: id, Name: strings.TrimSpace(req.Name)})
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
		log.Printf("encoding user response error: %v", err)
	}
}

func (srv *Server) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := srv.usersRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
