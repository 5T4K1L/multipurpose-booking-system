package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrInvalidCatalog     = errors.New("invalid booking catalog input")
	ErrCatalogNotFound    = errors.New("booking catalog item not found")
	ErrCatalogUnavailable = errors.New("booking catalog is temporarily unavailable")
)

// BookingCatalogStore is implemented by sqlc on the existing tenant transaction.
// All lookups require business_id as well as id; unscoped lookups are unavailable.
type BookingCatalogStore interface {
	BookingOptionsStore
	CatalogUpdateStore
	AvailabilityStore
	BookingOccupancyStore
	SchedulingStore
	CatalogListStore
	CatalogEligibilityStore
	CreateLocation(context.Context, db.CreateLocationParams) (db.Location, error)
	GetLocation(context.Context, db.GetLocationParams) (db.Location, error)
	CreateService(context.Context, db.CreateServiceParams) (db.Service, error)
	GetService(context.Context, db.GetServiceParams) (db.Service, error)
	CreateStaffProfile(context.Context, db.CreateStaffProfileParams) (db.StaffProfile, error)
	GetStaffProfile(context.Context, db.GetStaffProfileParams) (db.StaffProfile, error)
	CreateResource(context.Context, db.CreateResourceParams) (db.Resource, error)
	GetResource(context.Context, db.GetResourceParams) (db.Resource, error)
}
type BookingCatalogService struct{ tenants *BusinessService }

func NewBookingCatalogService(tenants *BusinessService) *BookingCatalogService {
	return &BookingCatalogService{tenants: tenants}
}

type LocationInput struct {
	Name     string
	Timezone string
}
type ServiceInput struct {
	AllDay          bool
	Name            string
	DurationMinutes int32
}
type StaffProfileInput struct {
	Name     string
	RoleName string
}
type ResourceInput struct {
	Name       string
	LocationID pgtype.UUID
	Capacity   int32
}

func catalogName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	return name, utf8.ValidString(name) && utf8.RuneCountInString(name) >= 1 && utf8.RuneCountInString(name) <= 100 && strings.IndexFunc(name, unicode.IsControl) < 0
}

// catalogOperation never returns a result before the enclosing transaction commits.
// Database errors remain internal; missing and other-tenant resources look alike.
func catalogOperation[T any](ctx context.Context, s *BookingCatalogService, actor int64, business pgtype.UUID, write bool, fn func(BookingCatalogStore, TenantAccess) (T, error)) (T, error) {
	var out T
	roles := []Role{RoleBusinessOwner, RoleBusinessAdmin}
	if !write {
		roles = append(roles, RoleManager, RoleStaff)
	}
	err := s.tenants.WithinTenant(ctx, actor, business, roles, func(q BusinessStore, a TenantAccess) error { var e error; out, e = fn(q, a); return e })
	if err != nil {
		var zero T
		if errors.Is(err, ErrInvalidCatalog) {
			return zero, ErrInvalidCatalog
		}
		if errors.Is(err, ErrCatalogConflict) {
			return zero, ErrCatalogConflict
		}
		if errors.Is(err, ErrForbidden) {
			return zero, ErrForbidden
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, ErrCatalogNotFound
		}
		return zero, ErrCatalogUnavailable
	}
	return out, nil
}
func (s *BookingCatalogService) CreateLocation(ctx context.Context, actor int64, business pgtype.UUID, in LocationInput) (db.Location, error) {
	name, ok := catalogName(in.Name)
	if !ok || len(in.Timezone) > 100 || in.Timezone == "" || in.Timezone == "Local" {
		return db.Location{}, ErrInvalidCatalog
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return db.Location{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Location, error) {
		return q.CreateLocation(ctx, db.CreateLocationParams{BusinessID: a.BusinessID(), Name: name, Timezone: in.Timezone})
	})
}
func (s *BookingCatalogService) CreateService(ctx context.Context, actor int64, business pgtype.UUID, in ServiceInput) (db.Service, error) {
	if in.AllDay {
		in.DurationMinutes = 1440
	}
	name, ok := catalogName(in.Name)
	if !ok || in.DurationMinutes < 1 || in.DurationMinutes > 10080 {
		return db.Service{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Service, error) {
		return q.CreateService(ctx, db.CreateServiceParams{BusinessID: a.BusinessID(), Name: name, DurationMinutes: in.DurationMinutes, AllDay: in.AllDay})
	})
}
func (s *BookingCatalogService) CreateStaffProfile(ctx context.Context, actor int64, business pgtype.UUID, in StaffProfileInput) (db.StaffProfile, error) {
	name, ok := catalogName(in.Name)
	role, valid := staffRoleLabel(in.RoleName)
	if !ok || !valid {
		return db.StaffProfile{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.StaffProfile, error) {
		return q.CreateStaffProfile(ctx, db.CreateStaffProfileParams{BusinessID: a.BusinessID(), Name: name, RoleName: role})
	})
}
func (s *BookingCatalogService) CreateResource(ctx context.Context, actor int64, business pgtype.UUID, in ResourceInput) (db.Resource, error) {
	name, ok := catalogName(in.Name)
	if !ok || !in.LocationID.Valid || in.Capacity < 1 {
		return db.Resource{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Resource, error) {
		if _, err := q.GetLocation(ctx, db.GetLocationParams{BusinessID: a.BusinessID(), ID: in.LocationID}); err != nil {
			return db.Resource{}, err
		}
		return q.CreateResource(ctx, db.CreateResourceParams{BusinessID: a.BusinessID(), LocationID: in.LocationID, Name: name, Capacity: in.Capacity})
	})
}
func (s *BookingCatalogService) GetLocation(ctx context.Context, actor int64, business, id pgtype.UUID) (db.Location, error) {
	if !id.Valid {
		return db.Location{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (db.Location, error) {
		return q.GetLocation(ctx, db.GetLocationParams{BusinessID: a.BusinessID(), ID: id})
	})
}
func (s *BookingCatalogService) GetService(ctx context.Context, actor int64, business, id pgtype.UUID) (db.Service, error) {
	if !id.Valid {
		return db.Service{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (db.Service, error) {
		return q.GetService(ctx, db.GetServiceParams{BusinessID: a.BusinessID(), ID: id})
	})
}
func (s *BookingCatalogService) GetStaffProfile(ctx context.Context, actor int64, business, id pgtype.UUID) (db.StaffProfile, error) {
	if !id.Valid {
		return db.StaffProfile{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (db.StaffProfile, error) {
		return q.GetStaffProfile(ctx, db.GetStaffProfileParams{BusinessID: a.BusinessID(), ID: id})
	})
}
func (s *BookingCatalogService) GetResource(ctx context.Context, actor int64, business, id pgtype.UUID) (db.Resource, error) {
	if !id.Valid {
		return db.Resource{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (db.Resource, error) {
		return q.GetResource(ctx, db.GetResourceParams{BusinessID: a.BusinessID(), ID: id})
	})
}

// Custom staff roles are business vocabulary, never authorization roles.
func staffRoleLabel(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	name, ok := catalogName(value)
	return name, ok && utf8.RuneCountInString(name) <= 60
}
