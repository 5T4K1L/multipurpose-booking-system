package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// HealthResponse contains the API health status.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler returns the current API health status.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Allow the local Next.js development server to call this endpoint.
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status: "ok",
	}

	// Return the health status as JSON.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}
