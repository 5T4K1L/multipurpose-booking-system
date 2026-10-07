package repository

import (
	"context"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

type AuthorizationRepository struct{ queries *db.Queries }

func NewAuthorizationRepository(queries *db.Queries) *AuthorizationRepository {
	return &AuthorizationRepository{queries: queries}
}
func (r *AuthorizationRepository) GetAuthorizationState(ctx context.Context, userID int64) (service.AuthorizationState, error) {
	rows, err := r.queries.GetAuthorizationState(ctx, userID)
	if err != nil {
		return service.AuthorizationState{}, err
	}
	state := service.AuthorizationState{}
	for _, row := range rows {
		state.Active = row.Status == "active"
		if row.Role != "" {
			state.PlatformRoles = append(state.PlatformRoles, service.Role(row.Role))
		}
	}
	return state, nil
}
