package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	port := ":8080"
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	fmt.Printf("Server started on %s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		panic(err)
	}
}
