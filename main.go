package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Route struct {
	Method func(w http.ResponseWriter, r *http.Request)
	Path   string
}

var users = []User{
	{ID: 1, Name: "Alice"},
	{ID: 2, Name: "Bob"},
	{ID: 3, Name: "Charlie"},
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}

func getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user ID", http.StatusBadRequest)
		return
	}

	for _, user := range users {
		if user.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	http.Error(w, "user not found", http.StatusNotFound)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	usersLen := len(users)

	limit := r.URL.Query().Get("limit")
	limitInt := usersLen
	if limit != "" {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil || limitInt <= 0 || limitInt > usersLen {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
	}

	resultUsers := users[0:limitInt]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultUsers)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
}

func main() {
	handlers := []Route{
		{helloHandler, "GET /"},
		{usersHandler, "GET /users"},
		{getUserByIDHandler, "GET /users/{id}"},
		{healthHandler, "GET /health"},
	}
	port := ":8080"
	mux := http.NewServeMux()

	for _, handler := range handlers {
		mux.HandleFunc(handler.Path, handler.Method)
	}

	fmt.Printf("Server started on %s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		panic(err)
	}
}
