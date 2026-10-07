package service

import (
	"context"
	"errors"
)

type Role string

const (
	RoleCustomer         Role = "CUSTOMER"
	RoleBusinessOwner    Role = "BUSINESS_OWNER"
	RoleBusinessAdmin    Role = "BUSINESS_ADMIN"
	RoleManager          Role = "MANAGER"
	RoleStaff            Role = "STAFF"
	RolePlatformAdmin    Role = "PLATFORM_ADMIN"
	RolePlatformOperator Role = "PLATFORM_OPERATOR"
)

var ErrForbidden = errors.New("access is forbidden")
var ErrAuthorizationUnavailable = errors.New("authorization is unavailable")

type AuthorizationState struct {
	Active        bool
	PlatformRoles []Role
}
type AuthorizationStore interface {
	GetAuthorizationState(context.Context, int64) (AuthorizationState, error)
}
type AuthorizationService struct{ store AuthorizationStore }

func NewAuthorizationService(store AuthorizationStore) *AuthorizationService {
	return &AuthorizationService{store: store}
}

// RequireAnyRole accepts only an authenticated server-derived user ID and a
// server-defined policy. Business roles deliberately fail closed until memberships
// exist. No implicit hierarchy or administrator bypass is provided.
func (s *AuthorizationService) RequireAnyRole(ctx context.Context, userID int64, allowed ...Role) error {
	if userID <= 0 || len(allowed) == 0 {
		return ErrForbidden
	}
	for _, role := range allowed {
		if role != RoleCustomer && role != RolePlatformAdmin && role != RolePlatformOperator {
			return ErrForbidden
		}
	}
	if s.store == nil {
		return ErrAuthorizationUnavailable
	}
	state, err := s.store.GetAuthorizationState(ctx, userID)
	if err != nil {
		return ErrAuthorizationUnavailable
	}
	if !state.Active {
		return ErrForbidden
	}
	for _, role := range state.PlatformRoles {
		if role != RolePlatformAdmin && role != RolePlatformOperator {
			return ErrForbidden
		}
	}
	for _, role := range allowed {
		if role == RoleCustomer {
			return nil
		}
		for _, grant := range state.PlatformRoles {
			if grant == role {
				return nil
			}
		}
	}
	return ErrForbidden
}

// RequireOwner checks resource ownership separately from role authorization.
// ownerID must come from a persisted resource, never directly from client input.
// Even a platform role does not bypass ownership.
func RequireOwner(authenticatedUserID, ownerID int64) error {
	if authenticatedUserID <= 0 || ownerID <= 0 || authenticatedUserID != ownerID {
		return ErrForbidden
	}
	return nil
}
