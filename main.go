package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type User struct {
	ID   int    `json:"id"`
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

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	usersMu.Lock()

	lastUserID := 0
	usersLen := len(users)

	if usersLen > 0 {
		lastUserID = users[usersLen-1].ID
	}

	user.ID = lastUserID + 1
	users = append(users, user)

	usersMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func main() {
	port := ":8080"
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users", createUserHandler)

	fmt.Printf("Server started on %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		panic(err)
	}
}
