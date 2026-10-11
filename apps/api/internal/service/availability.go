package service

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"sort"
	"time"
)

type AvailabilityStore interface {
	ListStaffAvailabilityWindows(context.Context, db.ListStaffAvailabilityWindowsParams) ([]db.ListStaffAvailabilityWindowsRow, error)
	DeleteStaffAvailabilityWindows(context.Context, db.DeleteStaffAvailabilityWindowsParams) error
	CreateStaffAvailabilityWindow(context.Context, db.CreateStaffAvailabilityWindowParams) error
	ListResourceAvailabilityWindows(context.Context, db.ListResourceAvailabilityWindowsParams) ([]db.ListResourceAvailabilityWindowsRow, error)
	DeleteResourceAvailabilityWindows(context.Context, db.DeleteResourceAvailabilityWindowsParams) error
	CreateResourceAvailabilityWindow(context.Context, db.CreateResourceAvailabilityWindowParams) error
	CheckSlotEligibility(context.Context, db.CheckSlotEligibilityParams) (db.CheckSlotEligibilityRow, error)
}
type AvailabilityWindow struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func normalizeWindows(in []AvailabilityWindow) ([]AvailabilityWindow, error) {
	if in == nil || len(in) > 32 {
		return nil, ErrInvalidCatalog
	}
	out := append([]AvailabilityWindow{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	for i := range out {
		w := &out[i]
		w.StartsAt = w.StartsAt.UTC()
		w.EndsAt = w.EndsAt.UTC()
		if w.StartsAt.Year() < 2000 || w.EndsAt.Year() > 2100 || !w.EndsAt.After(w.StartsAt) || w.EndsAt.Sub(w.StartsAt) > 7*24*time.Hour || w.StartsAt.Second() != 0 || w.EndsAt.Second() != 0 || w.StartsAt.Nanosecond() != 0 || w.EndsAt.Nanosecond() != 0 {
			return nil, ErrInvalidCatalog
		}
		if i > 0 && out[i-1].EndsAt.After(w.StartsAt) {
			return nil, ErrInvalidCatalog
		}
	}
	return out, nil
}
func availabilityEntity(ctx context.Context, q BookingCatalogStore, business, id pgtype.UUID, kind string) error {
	if kind == "staff" {
		_, e := q.GetStaffProfile(ctx, db.GetStaffProfileParams{BusinessID: business, ID: id})
		return e
	}
	_, e := q.GetResource(ctx, db.GetResourceParams{BusinessID: business, ID: id})
	return e
}
func availabilityWindows(ctx context.Context, q BookingCatalogStore, business, id pgtype.UUID, kind string) ([]AvailabilityWindow, error) {
	out := []AvailabilityWindow{}
	if kind == "staff" {
		rows, e := q.ListStaffAvailabilityWindows(ctx, db.ListStaffAvailabilityWindowsParams{BusinessID: business, StaffID: id})
		if e != nil {
			return nil, e
		}
		for _, w := range rows {
			out = append(out, AvailabilityWindow{w.StartsAt.Time.UTC(), w.EndsAt.Time.UTC()})
		}
	} else {
		rows, e := q.ListResourceAvailabilityWindows(ctx, db.ListResourceAvailabilityWindowsParams{BusinessID: business, ResourceID: id})
		if e != nil {
			return nil, e
		}
		for _, w := range rows {
			out = append(out, AvailabilityWindow{w.StartsAt.Time.UTC(), w.EndsAt.Time.UTC()})
		}
	}
	return out, nil
}
func (s *BookingCatalogService) GetAvailabilityWindows(ctx context.Context, actor int64, business, id pgtype.UUID, kind string) ([]AvailabilityWindow, error) {
	if !id.Valid || (kind != "staff" && kind != "resources") {
		return nil, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) ([]AvailabilityWindow, error) {
		if e := availabilityEntity(ctx, q, a.BusinessID(), id, kind); e != nil {
			return nil, e
		}
		return availabilityWindows(ctx, q, a.BusinessID(), id, kind)
	})
}
func (s *BookingCatalogService) PutAvailabilityWindows(ctx context.Context, actor int64, business, id pgtype.UUID, kind string, in []AvailabilityWindow) ([]AvailabilityWindow, error) {
	if !id.Valid || (kind != "staff" && kind != "resources") {
		return nil, ErrInvalidCatalog
	}
	windows, e := normalizeWindows(in)
	if e != nil {
		return nil, e
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) ([]AvailabilityWindow, error) {
		if e := availabilityEntity(ctx, q, a.BusinessID(), id, kind); e != nil {
			return nil, e
		}
		if kind == "staff" {
			if e := q.DeleteStaffAvailabilityWindows(ctx, db.DeleteStaffAvailabilityWindowsParams{BusinessID: a.BusinessID(), StaffID: id}); e != nil {
				return nil, e
			}
		} else {
			if e := q.DeleteResourceAvailabilityWindows(ctx, db.DeleteResourceAvailabilityWindowsParams{BusinessID: a.BusinessID(), ResourceID: id}); e != nil {
				return nil, e
			}
		}
		for _, w := range windows {
			start, end := pgtype.Timestamptz{Time: w.StartsAt, Valid: true}, pgtype.Timestamptz{Time: w.EndsAt, Valid: true}
			var e error
			if kind == "staff" {
				e = q.CreateStaffAvailabilityWindow(ctx, db.CreateStaffAvailabilityWindowParams{BusinessID: a.BusinessID(), StaffID: id, StartsAt: start, EndsAt: end})
			} else {
				e = q.CreateResourceAvailabilityWindow(ctx, db.CreateResourceAvailabilityWindowParams{BusinessID: a.BusinessID(), ResourceID: id, StartsAt: start, EndsAt: end})
			}
			if e != nil {
				return nil, e
			}
		}
		return windows, nil
	})
}
