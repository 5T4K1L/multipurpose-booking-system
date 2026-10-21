package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var ErrCatalogConflict = errors.New("catalog change conflicts with booking history")

type CatalogUpdateStore interface {
	CatalogHasUpcomingBookings(context.Context, db.CatalogHasUpcomingBookingsParams) (bool, error)
	ArchiveLocation(context.Context, db.ArchiveLocationParams) (int64, error)
	ArchiveService(context.Context, db.ArchiveServiceParams) (int64, error)
	ArchiveStaffProfile(context.Context, db.ArchiveStaffProfileParams) (int64, error)
	ArchiveResource(context.Context, db.ArchiveResourceParams) (int64, error)

	UpdateBusinessProfile(context.Context, db.UpdateBusinessProfileParams) (db.Business, error)
	UpdateLocation(context.Context, db.UpdateLocationParams) (db.Location, error)
	UpdateService(context.Context, db.UpdateServiceParams) (db.Service, error)
	UpdateStaffProfile(context.Context, db.UpdateStaffProfileParams) (db.StaffProfile, error)
	UpdateResource(context.Context, db.UpdateResourceParams) (db.Resource, error)
	ResourceHasBookings(context.Context, db.ResourceHasBookingsParams) (bool, error)
}

func validTimezone(zone string) bool {
	if zone == "" || zone == "Local" || len(zone) > 100 {
		return false
	}
	_, err := time.LoadLocation(zone)
	return err == nil
}
func (s *BusinessService) UpdateProfile(ctx context.Context, actor int64, id pgtype.UUID, in BusinessInput) (BusinessView, error) {
	name, ok := catalogName(in.Name)
	if !ok || !validTimezone(in.Timezone) {
		return BusinessView{}, ErrInvalidBusiness
	}
	var out BusinessView
	err := s.WithinTenant(ctx, actor, id, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		if name != a.business.Name {
			return ErrInvalidBusiness
		}
		b, err := q.UpdateBusinessProfile(ctx, db.UpdateBusinessProfileParams{ID: a.BusinessID(), Name: name, Timezone: in.Timezone})
		if err != nil {
			return err
		}
		out = BusinessView{b.ID, b.Name, b.Timezone, a.business.Role}
		return nil
	})
	if err != nil {
		return BusinessView{}, err
	}
	return out, nil
}
func (s *BookingCatalogService) UpdateLocation(ctx context.Context, actor int64, business, id pgtype.UUID, in LocationInput) (db.Location, error) {
	name, ok := catalogName(in.Name)
	if !ok || !id.Valid || !validTimezone(in.Timezone) {
		return db.Location{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Location, error) {
		return q.UpdateLocation(ctx, db.UpdateLocationParams{BusinessID: a.BusinessID(), ID: id, Name: name, Timezone: in.Timezone})
	})
}
func (s *BookingCatalogService) UpdateService(ctx context.Context, actor int64, business, id pgtype.UUID, in ServiceInput) (db.Service, error) {
	if in.AllDay {
		in.DurationMinutes = 1440
	}
	name, ok := catalogName(in.Name)
	if !ok || !id.Valid || in.DurationMinutes < 1 || in.DurationMinutes > 10080 {
		return db.Service{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Service, error) {
		return q.UpdateService(ctx, db.UpdateServiceParams{BusinessID: a.BusinessID(), ID: id, Name: name, DurationMinutes: in.DurationMinutes, AllDay: in.AllDay})
	})
}
func (s *BookingCatalogService) UpdateStaffProfile(ctx context.Context, actor int64, business, id pgtype.UUID, in StaffProfileInput) (db.StaffProfile, error) {
	name, ok := catalogName(in.Name)
	role, valid := staffRoleLabel(in.RoleName)
	if !valid || !ok || !id.Valid {
		return db.StaffProfile{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.StaffProfile, error) {
		return q.UpdateStaffProfile(ctx, db.UpdateStaffProfileParams{BusinessID: a.BusinessID(), ID: id, Name: name, RoleName: role})
	})
}
func (s *BookingCatalogService) UpdateResource(ctx context.Context, actor int64, business, id pgtype.UUID, in ResourceInput) (db.Resource, error) {
	name, ok := catalogName(in.Name)
	if !ok || !id.Valid || in.Capacity < 1 {
		return db.Resource{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (db.Resource, error) {
		old, err := q.GetResource(ctx, db.GetResourceParams{BusinessID: a.BusinessID(), ID: id})
		if err != nil {
			return db.Resource{}, err
		}
		if old.Capacity != in.Capacity {
			booked, err := q.ResourceHasBookings(ctx, db.ResourceHasBookingsParams{BusinessID: a.BusinessID(), ResourceID: id})
			if err != nil {
				return db.Resource{}, err
			}
			if booked {
				return db.Resource{}, ErrCatalogConflict
			}
		}
		return q.UpdateResource(ctx, db.UpdateResourceParams{BusinessID: a.BusinessID(), ID: id, Name: name, Capacity: in.Capacity})
	})
}

// Archive removes an item from the live catalog while retaining booking references.
// WithinTenant serializes this check and mutation with booking creation.
func (s *BookingCatalogService) Archive(ctx context.Context, actor int64, business, id pgtype.UUID, kind string) error {
	if !id.Valid || (kind != "locations" && kind != "services" && kind != "staff" && kind != "resources") {
		return ErrInvalidCatalog
	}
	_, err := catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (int64, error) {
		booked, e := q.CatalogHasUpcomingBookings(ctx, db.CatalogHasUpcomingBookingsParams{BusinessID: a.BusinessID(), ItemID: id, Kind: kind})
		if e != nil {
			return 0, e
		}
		if booked {
			return 0, ErrCatalogConflict
		}
		var count int64
		switch kind {
		case "locations":
			count, e = q.ArchiveLocation(ctx, db.ArchiveLocationParams{BusinessID: a.BusinessID(), ID: id})
		case "services":
			count, e = q.ArchiveService(ctx, db.ArchiveServiceParams{BusinessID: a.BusinessID(), ID: id})
		case "staff":
			count, e = q.ArchiveStaffProfile(ctx, db.ArchiveStaffProfileParams{BusinessID: a.BusinessID(), ID: id})
		case "resources":
			count, e = q.ArchiveResource(ctx, db.ArchiveResourceParams{BusinessID: a.BusinessID(), ID: id})
		}
		if e == nil && count == 0 {
			return 0, pgx.ErrNoRows
		}
		return count, e
	})
	return err
}
