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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var ErrBusinessAlreadyOwned = errors.New("this account already owns a business")

var ErrInvalidBusiness = errors.New("invalid business input")

type BusinessStore interface {
	GuestStore
	CustomerCRMStore
	CustomerFoundationStore
	BusinessSettingsStore
	BookingStore
	BookingCatalogStore
	GetUserByID(context.Context, int64) (db.User, error)
	CreateBusiness(context.Context, db.CreateBusinessParams) (db.Business, error)
	ListUserBusinesses(context.Context, db.ListUserBusinessesParams) ([]db.ListUserBusinessesRow, error)
	LockBusiness(context.Context, pgtype.UUID) (db.Business, error)
	LockBusinessAccess(context.Context, db.LockBusinessAccessParams) (db.LockBusinessAccessRow, error)
	GetBusinessMembership(context.Context, db.GetBusinessMembershipParams) (db.BusinessMembership, error)
	ListBusinessMemberships(context.Context, db.ListBusinessMembershipsParams) ([]db.BusinessMembership, error)
	PutBusinessMembership(context.Context, db.PutBusinessMembershipParams) error
	DeleteBusinessMembership(context.Context, db.DeleteBusinessMembershipParams) error
}
type BusinessTransactions interface {
	WithinBusiness(context.Context, func(BusinessStore) error) error
}
type BusinessService struct{ transactions BusinessTransactions }

func NewBusinessService(t BusinessTransactions) *BusinessService { return &BusinessService{t} }

type BusinessInput struct {
	Profile  *BusinessSettings `json:"profile,omitempty"`
	Name     string            `json:"name"`
	Timezone string            `json:"timezone"`
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
	if in.Profile != nil {
		v, e := normalizeBusinessSettings(*in.Profile)
		if e != nil {
			return BusinessView{}, e
		}
		in.Profile = &v
	}
	var out BusinessView
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if err := activeBusinessUser(ctx, q, actor); err != nil {
			return err
		}
		owned, err := q.OwnedBusinessExists(ctx, actor)
		if err != nil {
			return err
		}
		if owned {
			return ErrBusinessAlreadyOwned
		}
		b, err := q.CreateBusiness(ctx, db.CreateBusinessParams{Name: in.Name, Timezone: in.Timezone})
		if err != nil {
			return err
		}
		if err = q.PutBusinessMembership(ctx, db.PutBusinessMembershipParams{BusinessID: b.ID, UserID: actor, Role: string(RoleBusinessOwner)}); err != nil {
			return err
		}
		if in.Profile != nil {
			v := in.Profile
			if _, err = q.PutBusinessSettings(ctx, db.PutBusinessSettingsParams{BusinessID: b.ID, Tagline: v.Tagline, Description: v.Description, ContactEmail: v.ContactEmail, ContactPhone: v.ContactPhone, Website: v.Website}); err != nil {
				return err
			}
		}
		out = BusinessView{b.ID, b.Name, b.Timezone, RoleBusinessOwner}
		return nil
	})
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.ConstraintName == "one_owned_business_per_user" {
			return BusinessView{}, ErrBusinessAlreadyOwned
		}
		return BusinessView{}, err
	}
	return out, nil
}

// List retains the first-page convenience API; HTTP callers use ListPage.
func (s *BusinessService) List(ctx context.Context, actor int64) ([]BusinessView, error) {
	page, err := s.ListPage(ctx, actor, PageRequest{})
	return page.Items, err
}
func (s *BusinessService) within(ctx context.Context, actor int64, id pgtype.UUID, fn func(BusinessStore, BusinessView) error) error {
	return s.WithinTenant(ctx, actor, id, []Role{RoleBusinessOwner, RoleBusinessAdmin, RoleManager, RoleStaff}, func(q BusinessStore, a TenantAccess) error { return fn(q, a.business) })
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
	page, err := s.MembersPage(ctx, actor, id, PageRequest{})
	return page.Items, err
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
