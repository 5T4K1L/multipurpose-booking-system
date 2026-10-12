package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrBookingState    = errors.New("booking cannot be changed")
)

type BookingLifecycleStore interface {
	GetTenantBooking(context.Context, db.GetTenantBookingParams) (db.Booking, error)
	CloseBooking(context.Context, db.CloseBookingParams) (db.Booking, error)
	GetBookingChange(context.Context, db.GetBookingChangeParams) (db.BookingChange, error)
	CreateBookingChange(context.Context, db.CreateBookingChangeParams) error
	CreateReplacementBooking(context.Context, db.CreateReplacementBookingParams) (db.Booking, error)
}
type BookingChangeResult struct {
	Booking     BookingResult  `json:"booking"`
	Replacement *BookingResult `json:"replacement,omitempty"`
	Replayed    bool           `json:"replayed"`
}

func (s *BookingService) Cancel(ctx context.Context, actor int64, business, id pgtype.UUID, key string) (BookingChangeResult, error) {
	return s.change(ctx, actor, business, id, key, "cancel", time.Time{})
}
func (s *BookingService) Reschedule(ctx context.Context, actor int64, business, id pgtype.UUID, key string, start time.Time) (BookingChangeResult, error) {
	if start.Year() < 2000 || start.Year() > 2100 || start.Second() != 0 || start.Nanosecond() != 0 {
		return BookingChangeResult{}, ErrInvalidBooking
	}
	return s.change(ctx, actor, business, id, key, "reschedule", start.UTC())
}
func bookingChangeHash(action string, id pgtype.UUID, start time.Time) string {
	data, _ := json.Marshal(struct {
		Action   string
		ID       pgtype.UUID
		StartsAt time.Time
	}{action, id, start})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (s *BookingService) change(ctx context.Context, actor int64, business, id pgtype.UUID, key, action string, start time.Time) (BookingChangeResult, error) {
	if !id.Valid || !validBookingKey(key) {
		return BookingChangeResult{}, ErrInvalidBooking
	}
	hash := bookingChangeHash(action, id, start)
	var out BookingChangeResult
	err := s.tenants.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		prior, e := q.GetBookingChange(ctx, db.GetBookingChangeParams{BusinessID: a.BusinessID(), ActorID: a.ActorID(), IdempotencyKey: key})
		if e == nil {
			if prior.RequestHash != hash {
				return ErrIdempotencyConflict
			}
			if e := json.Unmarshal(prior.Response, &out); e != nil {
				return e
			}
			out.Replayed = true
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		old, e := q.GetTenantBooking(ctx, db.GetTenantBookingParams{BusinessID: a.BusinessID(), ID: id})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		if e != nil {
			return e
		}
		if old.Status != "confirmed" || !time.Now().Before(old.BusyStartsAt.Time) {
			return ErrBookingState
		}
		status := "cancelled"
		var params db.CreateBookingParams
		if action == "reschedule" {
			if start.Equal(old.StartsAt.Time) {
				return ErrInvalidBooking
			}
			// The reserved prefix cannot be supplied as a public creation key.
			internalKey := fmt.Sprintf("~reschedule:%x", id.Bytes)
			params, e = prepareBooking(ctx, q, a, internalKey, hash, BookingInput{old.ServiceID, old.LocationID, old.StaffID, old.ResourceID, start})
			if e != nil {
				return e
			}
			status = "rescheduled"
		}
		closed, e := q.CloseBooking(ctx, db.CloseBookingParams{BusinessID: a.BusinessID(), ID: id, Status: status, ClosedBy: pgtype.Int8{Int64: a.ActorID(), Valid: true}})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrBookingState
		}
		if e != nil {
			return e
		}
		out.Booking = bookingResult(closed, false)
		if action == "reschedule" {
			// Release the old occupancy and acquire the new one atomically. A failed
			// insert, audit write, or commit restores the original confirmed booking.
			replacement, e := q.CreateReplacementBooking(ctx, db.CreateReplacementBookingParams{
				BusinessID: params.BusinessID, ServiceID: params.ServiceID, LocationID: params.LocationID,
				StaffID: params.StaffID, ResourceID: params.ResourceID, CreatedBy: params.CreatedBy,
				IdempotencyKey: params.IdempotencyKey, RequestHash: params.RequestHash,
				StartsAt: params.StartsAt, EndsAt: params.EndsAt, BusyStartsAt: params.BusyStartsAt, BusyEndsAt: params.BusyEndsAt,
				SupersedesBookingID: id,
			})
			if e != nil {
				return e
			}
			v := bookingResult(replacement, false)
			out.Replacement = &v
		}
		response, e := json.Marshal(out)
		if e != nil {
			return e
		}
		return q.CreateBookingChange(ctx, db.CreateBookingChangeParams{BusinessID: a.BusinessID(), ActorID: a.ActorID(), IdempotencyKey: key, BookingID: id, Action: action, RequestHash: hash, Response: response})
	})
	if err != nil {
		return BookingChangeResult{}, bookingError(err)
	}
	return out, nil
}
