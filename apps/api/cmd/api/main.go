package main

import (
	"context"
	"log"
	"net/http"

	"multipurpose-booking-system/api/internal/database"
	"multipurpose-booking-system/api/internal/server"
)

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL before starting the API.
	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Register the API health endpoint.
	mux.HandleFunc("/health", server.HealthHandler)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("PostgreSQL connection verified")
	log.Println("Server is running on http://localhost:8080")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
