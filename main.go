package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UpdateUserRequest struct {
	Name string `json:"name"`
}

var (
	users []User = []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}
	usersMu sync.Mutex
)

func setContentTypeJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	usersMu.Lock()
	result := make([]User, len(users))
	copy(result, users)
	usersMu.Unlock()

	setContentTypeJSON(w)
	json.NewEncoder(w).Encode(result)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	func() {
		usersMu.Lock()
		defer usersMu.Unlock()

		lastUserID := 0
		usersLen := len(users)

		if usersLen > 0 {
			lastUserID = users[usersLen-1].ID
		}

		user.ID = lastUserID + 1
		users = append(users, user)
	}()

	setContentTypeJSON(w)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func editUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid User ID", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	err = func() error {
		usersMu.Lock()
		defer usersMu.Unlock()
		for i := range users {
			user := &users[i]
			if user.ID == userID {
				user.Name = req.Name
				return nil
			}
		}

		return errors.New("User not found")
	}()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	setContentTypeJSON(w)
	json.NewEncoder(w).Encode(req)
}

func main() {
	port := ":8080"
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users", getUsersHandler)
	mux.HandleFunc("POST /users", createUserHandler)
	mux.HandleFunc("PATCH /users/{id}", editUserHandler)

	fmt.Printf("Server started on %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		panic(err)
	}
}
