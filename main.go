package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Request struct {
	Method func(w http.ResponseWriter, r *http.Request)
	Path   string
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	handlers := []Request{
		{helloHandler, "/"},
		{usersHandler, "/users"},
		{healthHandler, "/health"},
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
