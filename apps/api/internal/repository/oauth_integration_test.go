package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"multipurpose-booking-system/api/internal/database"
	"multipurpose-booking-system/api/internal/db"
)

func TestOAuthRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	dbPool, err := database.Connect(ctx)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer dbPool.Close()

	queries := db.New(dbPool)

	userRepository := NewUserRepository(queries)
	oauthRepository := NewOAuthRepository(queries)

	email := "oauth-integration-test@example.com"
	username := "OAuth Integration Test"

	user, err := userRepository.CreateUser(
		ctx,
		email,
		username,
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		_, err := dbPool.Exec(
			ctx,
			"DELETE FROM users WHERE id = $1",
			user.ID,
		)
		if err != nil {
			t.Fatalf("delete test user: %v", err)
		}
	}()

	created, err := oauthRepository.CreateOAuthIdentity(
		ctx,
		user.ID,
		"google",
		"test-google-subject",
	)
	if err != nil {
		t.Fatalf("create OAuth identity: %v", err)
	}

	found, err := oauthRepository.GetOAuthIdentity(
		ctx,
		"google",
		"test-google-subject",
	)
	if err != nil {
		t.Fatalf("get OAuth identity: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf(
			"expected identity ID %d, got %d",
			created.ID,
			found.ID,
		)
	}

	if found.UserID != user.ID {
		t.Fatalf(
			"expected user ID %d, got %d",
			user.ID,
			found.UserID,
		)
	}

	if found.Provider != "google" {
		t.Fatalf(
			"expected provider google, got %q",
			found.Provider,
		)
	}

	if found.ProviderSubject != "test-google-subject" {
		t.Fatalf(
			"unexpected provider subject: %q",
			found.ProviderSubject,
		)
	}

	foundByUser, err := oauthRepository.GetUserOAuthIdentity(
		ctx,
		user.ID,
		"google",
	)
	if err != nil {
		t.Fatalf(
			"get OAuth identity by user: %v",
			err,
		)
	}

	if foundByUser.ID != created.ID {
		t.Fatalf(
			"expected user lookup identity ID %d, got %d",
			created.ID,
			foundByUser.ID,
		)
	}
}
