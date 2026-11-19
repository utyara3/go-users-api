package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type User struct {
	ID   int64
	Name string
}

type Server struct {
	userService userService
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

func newMux(srv *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", srv.APIHealthCheckHandler)
	mux.HandleFunc("GET /users/{id}", srv.GetUserHandler)
	mux.HandleFunc("POST /users", srv.CreateNewUserHandler)
	mux.HandleFunc("PUT /users/{id}", srv.UpdateUserHandler)
	mux.HandleFunc("DELETE /users/{id}", srv.DeleteUserHandler)
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encoding response error: %v", err)
	}
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	writeJSON(w, code, JSONError{Err: message})
}

func writeUser(w http.ResponseWriter, code int, user *User) {
	writeJSON(w, code, userResponse{ID: user.ID, Name: user.Name})
}

func (srv *Server) APIHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv *Server) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	input := GetUserInput{ID: userID}

	user, err := srv.userService.Get(r.Context(), input)
	if err != nil {

		if errors.Is(err, ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
		} else if errors.Is(err, ErrWrongUserID) {
			sendJSONError(w, "Wrong user ID", http.StatusBadRequest)
		} else {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeUser(w, http.StatusOK, user)
}

func (srv *Server) CreateNewUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	input := CreateUserInput{Name: req.Name}

	user, err := srv.userService.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrWrongUserName) {
			sendJSONError(w, "Wrong user name", http.StatusBadRequest)
		} else {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeUser(w, http.StatusCreated, user)
}

func (srv *Server) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
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

	input := UpdateUserInput{ID: id, Name: req.Name}

	updatedUser, err := srv.userService.Update(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrWrongUserID):
			sendJSONError(w, "Wrong user ID", http.StatusBadRequest)
		case errors.Is(err, ErrWrongUserName):
			sendJSONError(w, "Wrong user name", http.StatusBadRequest)
		case errors.Is(err, ErrUserNotFound):
			sendJSONError(w, "User not found", http.StatusNotFound)
		default:
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writeUser(w, http.StatusOK, updatedUser)
}

func (srv *Server) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	input := DeleteUserInput{ID: id}

	if err := srv.userService.Delete(r.Context(), input); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
		} else {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
