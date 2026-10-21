package service

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

type BookingOptionsStore interface {
	ListBookingReadiness(context.Context, db.ListBookingReadinessParams) ([]db.ListBookingReadinessRow, error)
	GetServiceBookingOptions(context.Context, db.GetServiceBookingOptionsParams) (db.GetServiceBookingOptionsRow, error)
	PutServiceBookingOptions(context.Context, db.PutServiceBookingOptionsParams) error
}

type BookingReadinessItem struct {
	Kind           string      `json:"kind"`
	ID             pgtype.UUID `json:"id"`
	Name           string      `json:"name"`
	Detail         string      `json:"detail"`
	NeedsAttention bool        `json:"needs_attention"`
}

func (s *BookingCatalogService) BookingReadiness(ctx context.Context, actor int64, business, id pgtype.UUID) ([]BookingReadinessItem, error) {
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) ([]BookingReadinessItem, error) {
		if _, e := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: id}); e != nil {
			return nil, e
		}
		rows, e := q.ListBookingReadiness(ctx, db.ListBookingReadinessParams{BusinessID: business, ID: id})
		if e != nil {
			return nil, e
		}
		if len(rows) > 1000 {
			return nil, ErrAvailabilityComplex
		}
		out := []BookingReadinessItem{}
		for _, row := range rows {
			out = append(out, BookingReadinessItem{row.Kind, row.ID, row.Name, row.Detail, row.NeedsAttention})
		}
		return out, nil
	})
}

type BookingOptions struct {
	Enabled        bool  `json:"enabled"`
	MinimumMinutes int32 `json:"minimum_minutes"`
	MaximumMinutes int32 `json:"maximum_minutes"`
}

func bookingOptions(row db.GetServiceBookingOptionsRow) BookingOptions {
	return BookingOptions{row.Enabled, row.MinimumMinutes, row.MaximumMinutes}
}
func (s *BookingCatalogService) GetBookingOptions(ctx context.Context, actor int64, business, id pgtype.UUID) (BookingOptions, error) {
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (BookingOptions, error) {
		row, e := q.GetServiceBookingOptions(ctx, db.GetServiceBookingOptionsParams{BusinessID: business, ID: id})
		return bookingOptions(row), e
	})
}
func (s *BookingCatalogService) PutBookingOptions(ctx context.Context, actor int64, business, id pgtype.UUID, in BookingOptions) (BookingOptions, error) {
	if !id.Valid || in.MinimumMinutes < 1 || in.MaximumMinutes < in.MinimumMinutes || in.MaximumMinutes > 10080 {
		return BookingOptions{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (BookingOptions, error) {
		offered, e := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: id})
		if e != nil {
			return BookingOptions{}, e
		}
		if offered.AllDay && in.Enabled {
			return BookingOptions{}, ErrInvalidCatalog
		}
		e = q.PutServiceBookingOptions(ctx, db.PutServiceBookingOptionsParams{BusinessID: business, ServiceID: id, Enabled: in.Enabled, MinimumMinutes: in.MinimumMinutes, MaximumMinutes: in.MaximumMinutes})
		return in, e
	})
}

// A duration supplied by a customer must be authorized by the current policy.
func customerDuration(ctx context.Context, q BookingCatalogStore, business, id pgtype.UUID, minutes int32) (int32, error) {
	offered, e := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: id})
	if e != nil {
		return 0, e
	}
	if offered.AllDay {
		if minutes < 1 || minutes > 1560 {
			return 0, ErrInvalidBooking
		}
		return minutes, nil
	}
	options, e := q.GetServiceBookingOptions(ctx, db.GetServiceBookingOptionsParams{BusinessID: business, ID: id})
	if e != nil {
		return 0, e
	}
	if options.Enabled {
		if minutes < options.MinimumMinutes || minutes > options.MaximumMinutes {
			return 0, ErrInvalidBooking
		}
	} else if minutes != offered.DurationMinutes {
		return 0, ErrPublicationChanged
	}
	return minutes, nil
}
