package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"multipurpose-booking-system/api/internal/database"
	"multipurpose-booking-system/api/internal/db"
)

func TestUserRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to the local PostgreSQL database.
	dbPool, err := database.Connect(ctx)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer dbPool.Close()

	// Use the generated sqlc queries for the repository.
	queries := db.New(dbPool)
	userRepository := NewUserRepository(queries)

	email := "integration-test@example.com"
	username := "Integration Test User"

	// Create temporary test data.
	createdUser, err := userRepository.CreateUser(ctx, email, username)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	// Remove the temporary user after the test.
	defer func() {
		_, err := dbPool.Exec(ctx, "DELETE FROM users WHERE id = $1", createdUser.ID)
		if err != nil {
			t.Fatalf("delete test user: %v", err)
		}
	}()

	// Read the user back through the repository.
	foundUser, err := userRepository.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("get test user by ID: %v", err)
	}

	if foundUser.ID != createdUser.ID {
		t.Fatalf("expected ID %d, got %d", createdUser.ID, foundUser.ID)
	}

	if foundUser.Email != email {
		t.Fatalf("expected email %q, got %q", email, foundUser.Email)
	}

	if foundUser.Username != username {
		t.Fatalf("expected username %q, got %q", username, foundUser.Username)
	}

	// Verify lookup by email works too.
	foundByEmail, err := userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get test user by email: %v", err)
	}

	if foundByEmail.ID != createdUser.ID {
		t.Fatalf(
			"expected email lookup ID %d, got %d",
			createdUser.ID,
			foundByEmail.ID,
		)
	}
}
