package repository

import (
	"context"
	"fmt"
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

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test transaction: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil {
			t.Errorf("roll back test data: %v", err)
		}
	}()
	queries := db.New(tx)

	userRepository := NewUserRepository(queries)
	oauthRepository := NewOAuthRepository(queries)

	email := fmt.Sprintf("oauth-integration-%d@example.com", time.Now().UnixNano())
	username := "OAuth Integration Test"

	user, err := userRepository.CreateUser(
		ctx,
		email,
		username,
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	created, err := oauthRepository.CreateOAuthIdentity(
		ctx,
		user.ID,
		"google",
		email,
	)
	if err != nil {
		t.Fatalf("create OAuth identity: %v", err)
	}

	found, err := oauthRepository.GetOAuthIdentity(
		ctx,
		"google",
		email,
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

	if found.ProviderSubject != email {
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
