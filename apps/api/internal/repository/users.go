package repository

import (
	"context"
	"fmt"

	"multipurpose-booking-system/api/internal/db"
)

// UserRepository provides database access for users.
type UserRepository struct {
	queries *db.Queries
}

// NewUserRepository creates a user repository from the sqlc query layer.
func NewUserRepository(queries *db.Queries) *UserRepository {
	return &UserRepository{
		queries: queries,
	}
}

// GetUserByID retrieves a user by ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return db.User{}, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return db.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

// CreateUser creates and returns a user.
func (r *UserRepository) CreateUser(
	ctx context.Context,
	email string,
	username string,
) (db.User, error) {
	user, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    email,
		Username: username,
	})
	if err != nil {
		return db.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
