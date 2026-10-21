package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type CustomerCancellation struct {
	CustomerBooking
	Replayed         bool `json:"replayed"`
	AlreadyCancelled bool `json:"already_cancelled"`
}

// Customer ownership is independent of membership, catalog and publication.
// Lock the business before reading lifecycle state, matching operator writers.
func (s *BusinessService) CancelCustomerBooking(ctx context.Context, actor int64, id pgtype.UUID, key string) (CustomerCancellation, error) {
	if !id.Valid || !validBookingKey(key) {
		return CustomerCancellation{}, ErrInvalidBooking
	}
	digest := sha256.Sum256([]byte(key))
	storedKey := "~customer-cancel:" + hex.EncodeToString(digest[:])
	hash := bookingChangeHash("customer-cancel", id, time.Time{})
	var out CustomerCancellation
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if e := activeBusinessUser(ctx, q, actor); e != nil {
			return e
		}
		business, e := q.LockCustomerBookingBusiness(ctx, db.LockCustomerBookingBusinessParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		if e != nil {
			return e
		}
		if e := activeBusinessUser(ctx, q, actor); e != nil {
			return e
		}
		b, e := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		if e != nil {
			return e
		}
		previous, e := q.GetBookingChange(ctx, db.GetBookingChangeParams{BusinessID: business, ActorID: actor, IdempotencyKey: storedKey})
		if e == nil {
			if previous.RequestHash != hash {
				return ErrIdempotencyConflict
			}
			if e := json.Unmarshal(previous.Response, &out); e != nil {
				return e
			}
			out.Replayed = true
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		out.CustomerBooking = customerBookingView(b)
		if b.Status == "cancelled" {
			out.AlreadyCancelled = true
			return nil
		}
		if b.Status != "confirmed" || !time.Now().Before(b.BusyStartsAt.Time) {
			return ErrBookingState
		}
		_, e = q.CloseBooking(ctx, db.CloseBookingParams{BusinessID: business, ID: id, Status: "cancelled", ClosedBy: pgtype.Int8{Int64: actor, Valid: true}})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrBookingState
		}
		if e != nil {
			return e
		}
		out.Status = "cancelled"
		response, e := json.Marshal(out)
		if e != nil {
			return e
		}
		return q.CreateBookingChange(ctx, db.CreateBookingChangeParams{BusinessID: business, ActorID: actor, IdempotencyKey: storedKey, BookingID: id, Action: "cancel", RequestHash: hash, Response: response})
	})
	if err != nil {
		return CustomerCancellation{}, bookingError(err)
	}
	return out, nil
}

func customerBookingView(b db.GetCustomerBookingRow) CustomerBooking {
	return CustomerBooking{b.BusinessName, b.ServiceName, b.LocationName, b.ID, b.BusinessID, b.ServiceID, b.LocationID, b.StartsAt.Time.UTC(), b.EndsAt.Time.UTC(), b.Status, b.SupersedesBookingID, b.ReplacementID}
}

type CustomerBookingDetail struct {
	CustomerBooking
	CanReschedule bool      `json:"can_reschedule"`
	CanCancel     bool      `json:"can_cancel"`
	CancelBefore  time.Time `json:"cancel_before"`
}
