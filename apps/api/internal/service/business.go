package service

import (
	"context"
	"errors"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var ErrInvalidBusiness = errors.New("invalid business input")

type BusinessStore interface {
	GetUserByID(context.Context, int64) (db.User, error)
	CreateBusiness(context.Context, db.CreateBusinessParams) (db.Business, error)
	ListUserBusinesses(context.Context, int64) ([]db.ListUserBusinessesRow, error)
	LockBusinessAccess(context.Context, db.LockBusinessAccessParams) (db.LockBusinessAccessRow, error)
	GetBusinessMembership(context.Context, db.GetBusinessMembershipParams) (db.BusinessMembership, error)
	ListBusinessMemberships(context.Context, pgtype.UUID) ([]db.BusinessMembership, error)
	PutBusinessMembership(context.Context, db.PutBusinessMembershipParams) error
	DeleteBusinessMembership(context.Context, db.DeleteBusinessMembershipParams) error
}
type BusinessTransactions interface {
	WithinBusiness(context.Context, func(BusinessStore) error) error
}
type BusinessService struct{ transactions BusinessTransactions }

func NewBusinessService(t BusinessTransactions) *BusinessService { return &BusinessService{t} }

type BusinessInput struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}
type BusinessView struct {
	ID       pgtype.UUID `json:"id"`
	Name     string      `json:"name"`
	Timezone string      `json:"timezone"`
	Role     Role        `json:"role"`
}
type MembershipView struct {
	UserID int64 `json:"user_id"`
	Role   Role  `json:"role"`
}

func businessRole(r Role) bool {
	return r == RoleBusinessOwner || r == RoleBusinessAdmin || r == RoleManager || r == RoleStaff
}
func activeBusinessUser(ctx context.Context, q BusinessStore, id int64) error {
	if id <= 0 {
		return ErrForbidden
	}
	u, err := q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if u.Status != "active" {
		return ErrForbidden
	}
	return nil
}
func (s *BusinessService) Create(ctx context.Context, actor int64, in BusinessInput) (BusinessView, error) {
	in.Name = strings.TrimSpace(in.Name)
	if !utf8.ValidString(in.Name) || utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 100 || strings.IndexFunc(in.Name, unicode.IsControl) >= 0 || in.Timezone == "" || in.Timezone == "Local" {
		return BusinessView{}, ErrInvalidBusiness
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return BusinessView{}, ErrInvalidBusiness
	}
	var out BusinessView
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if err := activeBusinessUser(ctx, q, actor); err != nil {
			return err
		}
		b, err := q.CreateBusiness(ctx, db.CreateBusinessParams{Name: in.Name, Timezone: in.Timezone})
		if err != nil {
			return err
		}
		if err = q.PutBusinessMembership(ctx, db.PutBusinessMembershipParams{BusinessID: b.ID, UserID: actor, Role: string(RoleBusinessOwner)}); err != nil {
			return err
		}
		out = BusinessView{b.ID, b.Name, b.Timezone, RoleBusinessOwner}
		return nil
	})
	if err != nil {
		return BusinessView{}, err
	}
	return out, nil
}
func (s *BusinessService) List(ctx context.Context, actor int64) ([]BusinessView, error) {
	out := []BusinessView{}
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if err := activeBusinessUser(ctx, q, actor); err != nil {
			return err
		}
		rows, err := q.ListUserBusinesses(ctx, actor)
		if err != nil {
			return err
		}
		for _, b := range rows {
			if !businessRole(Role(b.Role)) {
				return ErrForbidden
			}
			out = append(out, BusinessView{b.ID, b.Name, b.Timezone, Role(b.Role)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Access is resolved from current membership inside every transaction. The UUID
// supplied by a client is only a selector, never a tenant authorization claim.
func (s *BusinessService) within(ctx context.Context, actor int64, id pgtype.UUID, fn func(BusinessStore, BusinessView) error) error {
	if actor <= 0 || !id.Valid {
		return ErrForbidden
	}
	return s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		b, err := q.LockBusinessAccess(ctx, db.LockBusinessAccessParams{ID: id, UserID: actor})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if !businessRole(Role(b.Role)) {
			return ErrForbidden
		}
		return fn(q, BusinessView{b.ID, b.Name, b.Timezone, Role(b.Role)})
	})
}
func (s *BusinessService) Get(ctx context.Context, actor int64, id pgtype.UUID) (BusinessView, error) {
	var out BusinessView
	err := s.within(ctx, actor, id, func(_ BusinessStore, b BusinessView) error { out = b; return nil })
	if err != nil {
		return BusinessView{}, err
	}
	return out, nil
}
func (s *BusinessService) Members(ctx context.Context, actor int64, id pgtype.UUID) ([]MembershipView, error) {
	out := []MembershipView{}
	err := s.within(ctx, actor, id, func(q BusinessStore, b BusinessView) error {
		if b.Role != RoleBusinessOwner && b.Role != RoleBusinessAdmin {
			return ErrForbidden
		}
		rows, err := q.ListBusinessMemberships(ctx, id)
		if err != nil {
			return err
		}
		for _, m := range rows {
			out = append(out, MembershipView{m.UserID, Role(m.Role)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Owner transfer/removal is intentionally unavailable in this milestone.
func (s *BusinessService) ChangeMember(ctx context.Context, actor int64, id pgtype.UUID, target int64, role Role, remove bool) error {
	if target <= 0 || (!remove && role != RoleBusinessAdmin && role != RoleManager && role != RoleStaff) {
		return ErrInvalidBusiness
	}
	return s.within(ctx, actor, id, func(q BusinessStore, b BusinessView) error {
		if b.Role != RoleBusinessOwner {
			return ErrForbidden
		}
		m, err := q.GetBusinessMembership(ctx, db.GetBusinessMembershipParams{BusinessID: id, UserID: target})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil && m.Role == string(RoleBusinessOwner) {
			return ErrForbidden
		}
		if remove {
			return q.DeleteBusinessMembership(ctx, db.DeleteBusinessMembershipParams{BusinessID: id, UserID: target})
		}
		if err := activeBusinessUser(ctx, q, target); err != nil {
			return err
		}
		return q.PutBusinessMembership(ctx, db.PutBusinessMembershipParams{BusinessID: id, UserID: target, Role: string(role)})
	})
}
