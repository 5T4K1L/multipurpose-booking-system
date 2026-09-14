package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"multipurpose-booking-system/api/internal/database"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/migrations"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/server"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Use the repository migration directory by default.
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "../../migrations"
	}

	// Apply pending database migrations before serving requests.
	if err := migrations.Run(databaseURL, migrationsPath); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	// Connect to PostgreSQL after migrations succeed.
	dbPool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	// Create the generated sqlc query layer.
	queries := db.New(dbPool)

	// Create the user repository.
	_ = repository.NewUserRepository(queries)

	mux := http.NewServeMux()

	// Register the API health endpoint.
	mux.HandleFunc("/health", server.HealthHandler)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Database migrations verified")
	log.Println("PostgreSQL connection verified")
	log.Println("Server is running on http://localhost:8080")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
