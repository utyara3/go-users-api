package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/utyara3/go-users-api/internal/user"
)

type Server struct {
	userService *user.UserService
	logger      *slog.Logger
}

// NewServer wires an HTTP server around the user service. Fields are
// unexported on purpose: the server is configured through this constructor,
// not by setting struct fields from other packages.
func NewServer(userService *user.UserService, logger *slog.Logger) *Server {
	return &Server{
		userService: userService,
		logger:      logger,
	}
}

type JSONError struct {
	Err string `json:"error"`
}

type createUserRequest struct {
	Name string `json:"name"`
}

type changeUserNameRequest struct {
	Name string `json:"name"`
}

type userResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

var ErrInvalidJSON = errors.New("invalid JSON")

func NewMux(srv *Server) *http.ServeMux {
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

func (srv *Server) writeUser(w http.ResponseWriter, code int, user *user.User) {
	srv.writeJSON(w, code, userResponse{ID: user.ID, Name: user.Name})
}

func (srv *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		srv.sendJSONError(w, "User not found", http.StatusNotFound)
	case errors.Is(err, user.ErrWrongUserID):
		srv.sendJSONError(w, "Wrong user ID", http.StatusBadRequest)
	case errors.Is(err, user.ErrWrongUserName):
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
		srv.handleError(w, r, user.ErrWrongUserID)
		return
	}

	input := user.GetUserInput{ID: userID}

	user, err := srv.userService.Get(r.Context(), input)
	if err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeUser(w, http.StatusOK, user)
}

func (srv *Server) CreateNewUserHandler(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		srv.handleError(w, r, ErrInvalidJSON)
		return
	}

	input := user.CreateUserInput{Name: req.Name}

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
		srv.handleError(w, r, user.ErrWrongUserID)
		return
	}

	var req changeUserNameRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		srv.handleError(w, r, ErrInvalidJSON)
		return
	}

	input := user.UpdateUserInput{ID: id, Name: req.Name}

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
		srv.handleError(w, r, user.ErrWrongUserID)
		return
	}

	input := user.DeleteUserInput{ID: id}

	if err := srv.userService.Delete(r.Context(), input); err != nil {
		srv.handleError(w, r, err)
		return
	}

	srv.writeJSON(w, http.StatusNoContent, nil)
}
