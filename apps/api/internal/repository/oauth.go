package repository

import (
	"context"
	"fmt"

	"multipurpose-booking-system/api/internal/db"
)

// OAuthRepository provides database access for OAuth identities.
type OAuthRepository struct {
	queries *db.Queries
}

// NewOAuthRepository creates an OAuth repository.
func NewOAuthRepository(queries *db.Queries) *OAuthRepository {
	return &OAuthRepository{
		queries: queries,
	}
}

// GetOAuthIdentity retrieves an identity by provider and stable subject.
func (r *OAuthRepository) GetOAuthIdentity(
	ctx context.Context,
	provider string,
	providerSubject string,
) (db.UserOauthIdentity, error) {
	identity, err := r.queries.GetOAuthIdentity(
		ctx,
		db.GetOAuthIdentityParams{
			Provider:        provider,
			ProviderSubject: providerSubject,
		},
	)
	if err != nil {
		return db.UserOauthIdentity{}, fmt.Errorf(
			"get OAuth identity: %w",
			err,
		)
	}

	return identity, nil
}

// GetUserOAuthIdentity retrieves a user's identity for a provider.
func (r *OAuthRepository) GetUserOAuthIdentity(
	ctx context.Context,
	userID int64,
	provider string,
) (db.UserOauthIdentity, error) {
	identity, err := r.queries.GetUserOAuthIdentity(
		ctx,
		db.GetUserOAuthIdentityParams{
			UserID:   userID,
			Provider: provider,
		},
	)
	if err != nil {
		return db.UserOauthIdentity{}, fmt.Errorf(
			"get user OAuth identity: %w",
			err,
		)
	}

	return identity, nil
}

// CreateOAuthIdentity explicitly links an external identity to a local user.
func (r *OAuthRepository) CreateOAuthIdentity(
	ctx context.Context,
	userID int64,
	provider string,
	providerSubject string,
) (db.UserOauthIdentity, error) {
	identity, err := r.queries.CreateOAuthIdentity(
		ctx,
		db.CreateOAuthIdentityParams{
			UserID:          userID,
			Provider:        provider,
			ProviderSubject: providerSubject,
		},
	)
	if err != nil {
		return db.UserOauthIdentity{}, fmt.Errorf(
			"create OAuth identity: %w",
			err,
		)
	}

	return identity, nil
}

// DeleteOAuthIdentity removes an identity only when it belongs to the user.
func (r *OAuthRepository) DeleteOAuthIdentity(
	ctx context.Context,
	userID int64,
	identityID int64,
) error {
	if err := r.queries.DeleteOAuthIdentity(
		ctx,
		db.DeleteOAuthIdentityParams{
			ID:     identityID,
			UserID: userID,
		},
	); err != nil {
		return fmt.Errorf(
			"delete OAuth identity: %w",
			err,
		)
	}

	return nil
}
