package main

import (
	"context"
	"log"
	"multipurpose-booking-system/api/internal/database"
	"net/http"
)

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL before starting the server
	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
	}
}
