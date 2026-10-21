package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type CustomerFoundationStore interface {
	LockCustomerBookingBusiness(context.Context, db.LockCustomerBookingBusinessParams) (pgtype.UUID, error)
	ListCustomerBookings(context.Context, db.ListCustomerBookingsParams) ([]db.ListCustomerBookingsRow, error)
	CreateCustomerBooking(context.Context, db.CreateCustomerBookingParams) (db.Booking, error)
	BusinessPublicationStore
	SetBusinessPublication(context.Context, db.SetBusinessPublicationParams) (bool, error)
	GetCustomerBooking(context.Context, db.GetCustomerBookingParams) (db.GetCustomerBookingRow, error)
}

// Ownership is independent of business membership and publication. The actor is
// supplied by trusted authentication, never by a request body or query parameter.
type CustomerBooking struct {
	BusinessName  string      `json:"business_name"`
	ServiceName   string      `json:"service_name"`
	LocationName  string      `json:"location_name"`
	ID            pgtype.UUID `json:"id"`
	BusinessID    pgtype.UUID `json:"business_id"`
	ServiceID     pgtype.UUID `json:"service_id"`
	LocationID    pgtype.UUID `json:"location_id"`
	StartsAt      time.Time   `json:"starts_at"`
	EndsAt        time.Time   `json:"ends_at"`
	Status        string      `json:"status"`
	SupersedesID  pgtype.UUID `json:"supersedes_booking_id"`
	ReplacementID pgtype.UUID `json:"replacement_id"`
}

func (s *BusinessService) CustomerBooking(ctx context.Context, actor int64, id pgtype.UUID) (CustomerBookingDetail, error) {
	var out CustomerBookingDetail
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if err := activeBusinessUser(ctx, q, actor); err != nil {
			return err
		}
		b, err := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
		if err != nil {
			return err
		}
		out = CustomerBookingDetail{CustomerBooking: customerBookingView(b), CanReschedule: b.Status == "confirmed" && time.Now().Before(b.BusyStartsAt.Time), CanCancel: b.Status == "confirmed" && time.Now().Before(b.BusyStartsAt.Time), CancelBefore: b.BusyStartsAt.Time.UTC()}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerBookingDetail{}, ErrBookingNotFound
	}
	if err != nil {
		return CustomerBookingDetail{}, err
	}
	return out, nil
}

// History stays accessible when a business unpublishes or archives its catalog.
func (s *BusinessService) CustomerBookings(ctx context.Context, actor int64, in PageRequest) (Page[CustomerBooking], error) {
	limit, c, e := parsePage(in, "customer-bookings", actor, pgtype.UUID{})
	if e != nil {
		return Page[CustomerBooking]{}, e
	}
	out := Page[CustomerBooking]{Items: []CustomerBooking{}}
	e = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if e := activeBusinessUser(ctx, q, actor); e != nil {
			return e
		}
		rows, e := q.ListCustomerBookings(ctx, db.ListCustomerBookingsParams{CustomerID: pgtype.Int8{Int64: actor, Valid: true}, HasCursor: in.Cursor != "", BeforeCreatedAt: pgtype.Timestamptz{Time: c.Created, Valid: in.Cursor != ""}, BeforeID: c.ID, PageSize: int32(limit + 1)})
		if e != nil {
			return e
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		for _, b := range rows {
			out.Items = append(out.Items, CustomerBooking{b.BusinessName, b.ServiceName, b.LocationName, b.ID, b.BusinessID, b.ServiceID, b.LocationID, b.StartsAt.Time.UTC(), b.EndsAt.Time.UTC(), b.Status, b.SupersedesBookingID, b.ReplacementID})
		}
		if more {
			last := rows[len(rows)-1]
			c.Created = last.CreatedAt.Time
			c.ID = last.ID
			out.NextCursor = encodeCursor(c)
		}
		return nil
	})
	if e != nil {
		return Page[CustomerBooking]{}, e
	}
	return out, nil
}
