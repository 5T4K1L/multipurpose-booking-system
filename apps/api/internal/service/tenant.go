package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

// TenantAccess is a snapshot resolved inside WithinTenant. Its fields cannot be
// supplied by a handler. It is not a bearer credential and must not be cached.
// Future resource operations must use the callback's transaction and BusinessID
// in every query; possession of this value outside the callback grants nothing.
type TenantAccess struct {
	actor    int64
	business BusinessView
}

func (a TenantAccess) ActorID() int64          { return a.actor }
func (a TenantAccess) BusinessID() pgtype.UUID { return a.business.ID }
func (a TenantAccess) Role() Role              { return a.business.Role }

// WithinTenant revalidates active membership and exact allowed roles, then runs
// work in the same transaction. An empty/unknown policy fails closed. Callbacks
// must not publish results or external side effects before this method succeeds.
// All membership writers must acquire the business lock first, as this method does.
func (s *BusinessService) WithinTenant(ctx context.Context, actor int64, id pgtype.UUID, allowed []Role, fn func(BusinessStore, TenantAccess) error) error {
	if actor <= 0 || !id.Valid || len(allowed) == 0 || fn == nil {
		return ErrForbidden
	}
	policy := append([]Role(nil), allowed...)
	for _, r := range policy {
		if !businessRole(r) {
			return ErrForbidden
		}
	}
	return s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		// A separate statement after this lock observes membership changes committed
		// while we waited. It also avoids inconsistent joined-row lock ordering.
		if _, err := q.LockBusiness(ctx, id); errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		} else if err != nil {
			return err
		}
		b, err := q.LockBusinessAccess(ctx, db.LockBusinessAccessParams{ID: id, UserID: actor})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		for _, r := range policy {
			if r == Role(b.Role) {
				return fn(q, TenantAccess{actor: actor, business: BusinessView{b.ID, b.Name, b.Timezone, Role(b.Role)}})
			}
		}
		return ErrForbidden
	})
}
