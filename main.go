package main

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"net/http"
)

type HealthStatus struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(HealthStatus{Status: "unavailable"})
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
