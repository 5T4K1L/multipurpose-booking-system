package main

import (
	"context"
	"encoding/json"
	"log"
	"multipurpose-booking-system/api/internal/database"
	"net/http"
)

// HealthResponse contains the API health status
type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL before starting the server
	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Health endpoint used to verify that the API is running and healthy
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
	}
}
