package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type User struct {
	ID   int64
	Name string
}

type Server struct {
	userService userService
	logger      *slog.Logger
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

type userResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

var ErrInvalidJSON = errors.New("invalid JSON")

func newMux(srv *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", srv.APIHealthCheckHandler)
	mux.HandleFunc("GET /users/{id}", srv.GetUserHandler)
	mux.HandleFunc("POST /users", srv.CreateNewUserHandler)
	mux.HandleFunc("PUT /users/{id}", srv.UpdateUserHandler)
	mux.HandleFunc("DELETE /users/{id}", srv.DeleteUserHandler)
	return mux
}

func (srv *Server) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		srv.logger.Error("encoding response error", slog.String("error", err.Error()))
	}
}

func (srv *Server) sendJSONError(w http.ResponseWriter, message string, code int) {
	srv.writeJSON(w, code, JSONError{Err: message})
}

func (srv *Server) writeUser(w http.ResponseWriter, code int, user *User) {
	srv.writeJSON(w, code, userResponse{ID: user.ID, Name: user.Name})
}

func (srv *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		srv.sendJSONError(w, "User not found", http.StatusNotFound)
	case errors.Is(err, ErrWrongUserID):
		srv.sendJSONError(w, "Wrong user ID", http.StatusBadRequest)
	case errors.Is(err, ErrWrongUserName):
		srv.sendJSONError(w, "Wrong user name", http.StatusBadRequest)
	case errors.Is(err, ErrInvalidJSON):
		srv.sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
	default:
		srv.logger.ErrorContext(r.Context(), "Internal server error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		srv.sendJSONError(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (srv *Server) APIHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	srv.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv *Server) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		srv.handleError(w, r, ErrWrongUserID)
		return
	}

	input := GetUserInput{ID: userID}

	user, err := srv.userService.Get(r.Context(), input)
	if err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeUser(w, http.StatusOK, user)
}

func (srv *Server) CreateNewUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		srv.handleError(w, r, ErrInvalidJSON)
		return
	}

	input := CreateUserInput{Name: req.Name}

	user, err := srv.userService.Create(r.Context(), input)
	if err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeUser(w, http.StatusCreated, user)
}

func (srv *Server) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		srv.handleError(w, r, ErrWrongUserID)
		return
	}

	var req ChangeUserNameRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		srv.handleError(w, r, ErrInvalidJSON)
		return
	}

	input := UpdateUserInput{ID: id, Name: req.Name}

	updatedUser, err := srv.userService.Update(r.Context(), input)
	if err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeUser(w, http.StatusOK, updatedUser)
}

func (srv *Server) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		srv.handleError(w, r, ErrWrongUserID)
		return
	}

	input := DeleteUserInput{ID: id}

	if err := srv.userService.Delete(r.Context(), input); err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeJSON(w, http.StatusNoContent, nil)
}
