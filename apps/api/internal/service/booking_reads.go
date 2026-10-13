package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type BookingReadStore interface {
	GetBookingView(context.Context, db.GetBookingViewParams) (db.GetBookingViewRow, error)
	ListTenantBookings(context.Context, db.ListTenantBookingsParams) ([]db.ListTenantBookingsRow, error)
}
type BookingView struct {
	ID pgtype.UUID `json:"id"`
	BookingInput
	EndsAt        time.Time   `json:"ends_at"`
	BusyStartsAt  time.Time   `json:"busy_starts_at"`
	BusyEndsAt    time.Time   `json:"busy_ends_at"`
	Status        string      `json:"status"`
	SupersedesID  pgtype.UUID `json:"supersedes_booking_id"`
	ReplacementID pgtype.UUID `json:"replacement_id"`
}

func publicBookingView(b db.GetBookingViewRow) BookingView {
	return BookingView{ID: b.ID, BookingInput: BookingInput{b.ServiceID, b.LocationID, b.StaffID, b.ResourceID, b.StartsAt.Time.UTC()}, EndsAt: b.EndsAt.Time.UTC(), BusyStartsAt: b.BusyStartsAt.Time.UTC(), BusyEndsAt: b.BusyEndsAt.Time.UTC(), Status: b.Status, SupersedesID: b.SupersedesBookingID, ReplacementID: b.ReplacementID}
}
func bookingRead[T any](ctx context.Context, s *BookingService, actor int64, business pgtype.UUID, fn func(BusinessStore, TenantAccess) (T, error)) (T, error) {
	var out T
	err := s.tenants.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin, RoleManager, RoleStaff}, func(q BusinessStore, a TenantAccess) error { var e error; out, e = fn(q, a); return e })
	if err != nil {
		var zero T
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, ErrBookingNotFound
		}
		return zero, bookingError(err)
	}
	return out, nil
}
func (s *BookingService) Get(ctx context.Context, actor int64, business, id pgtype.UUID) (BookingView, error) {
	if !id.Valid {
		return BookingView{}, ErrInvalidBooking
	}
	return bookingRead(ctx, s, actor, business, func(q BusinessStore, a TenantAccess) (BookingView, error) {
		v, e := q.GetBookingView(ctx, db.GetBookingViewParams{BusinessID: a.BusinessID(), ID: id})
		return publicBookingView(v), e
	})
}
func (s *BookingService) List(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest) (Page[BookingView], error) {
	limit, c, e := parseCatalogPage(in, "Bookings", actor, business, pgtype.UUID{})
	if e != nil {
		return Page[BookingView]{}, e
	}
	return bookingRead(ctx, s, actor, business, func(q BusinessStore, a TenantAccess) (Page[BookingView], error) {
		rows, e := q.ListTenantBookings(ctx, db.ListTenantBookingsParams{BusinessID: a.BusinessID(), HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(limit + 1)})
		if e != nil {
			return Page[BookingView]{}, e
		}
		views := make([]BookingView, 0, len(rows))
		for _, v := range rows {
			views = append(views, publicBookingView(db.GetBookingViewRow(v)))
		}
		return finishCatalogPage(views, limit, c, func(v BookingView) pgtype.UUID { return v.ID }), nil
	})
}
