package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type CustomerReschedule struct {
	Booking     CustomerBooking `json:"booking"`
	Replacement CustomerBooking `json:"replacement"`
	Replayed    bool            `json:"replayed"`
}

// Resolve customer ownership under the same business lock as all booking writers.
func ownedBookingForChange(ctx context.Context, q BusinessStore, actor int64, id pgtype.UUID) (db.GetCustomerBookingRow, error) {
	if e := activeBusinessUser(ctx, q, actor); e != nil {
		return db.GetCustomerBookingRow{}, e
	}
	_, e := q.LockCustomerBookingBusiness(ctx, db.LockCustomerBookingBusinessParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
	if errors.Is(e, pgx.ErrNoRows) {
		return db.GetCustomerBookingRow{}, ErrBookingNotFound
	}
	if e != nil {
		return db.GetCustomerBookingRow{}, e
	}
	if e = activeBusinessUser(ctx, q, actor); e != nil {
		return db.GetCustomerBookingRow{}, e
	}
	b, e := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
	if errors.Is(e, pgx.ErrNoRows) {
		return b, ErrBookingNotFound
	}
	return b, e
}
func customerMoveAllowed(ctx context.Context, q BusinessStore, b db.GetCustomerBookingRow) error {
	if b.Status != "confirmed" || !time.Now().Before(b.BusyStartsAt.Time) {
		return ErrBookingState
	}
	if _, e := q.LockBusiness(ctx, b.BusinessID); e != nil {
		return e
	}
	return nil
}

// No membership or publication is needed to manage an existing owned booking.
// Keep its exact duration and assignments; revalidate current catalog/hours/buffers.
func (s *BusinessService) RescheduleCustomerBooking(ctx context.Context, actor int64, id pgtype.UUID, key string, start time.Time) (CustomerReschedule, error) {
	if !id.Valid || !validBookingKey(key) || start.Year() < 2000 || start.Year() > 2100 || start.Second() != 0 || start.Nanosecond() != 0 {
		return CustomerReschedule{}, ErrInvalidBooking
	}
	start = start.UTC()
	digest := sha256.Sum256([]byte(key))
	storedKey := "~customer-move:" + hex.EncodeToString(digest[:])
	hash := bookingChangeHash("customer-move", id, start)
	var out CustomerReschedule
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		b, e := ownedBookingForChange(ctx, q, actor, id)
		if e != nil {
			return e
		}
		prior, e := q.GetBookingChange(ctx, db.GetBookingChangeParams{BusinessID: b.BusinessID, ActorID: actor, IdempotencyKey: storedKey})
		if e == nil {
			if prior.RequestHash != hash {
				return ErrIdempotencyConflict
			}
			if e = json.Unmarshal(prior.Response, &out); e != nil {
				return e
			}
			out.Replayed = true
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e = customerMoveAllowed(ctx, q, b); e != nil {
			return e
		}
		if start.Equal(b.StartsAt.Time) {
			return ErrInvalidBooking
		}
		a := TenantAccess{actor: actor, business: BusinessView{ID: b.BusinessID}}
		params, e := prepareBooking(ctx, q, a, fmt.Sprintf("~customer-move:%x", id.Bytes), hash, BookingInput{b.ServiceID, b.LocationID, b.StaffID, b.ResourceID, start}, int32(b.EndsAt.Time.Sub(b.StartsAt.Time)/time.Minute))
		if e != nil {
			return e
		}
		if _, e = q.CloseBooking(ctx, db.CloseBookingParams{BusinessID: b.BusinessID, ID: id, Status: "rescheduled", ClosedBy: pgtype.Int8{Int64: actor, Valid: true}}); e != nil {
			return e
		}
		replacement, e := q.CreateReplacementBooking(ctx, db.CreateReplacementBookingParams{BusinessID: params.BusinessID, ServiceID: params.ServiceID, LocationID: params.LocationID, StaffID: params.StaffID, ResourceID: params.ResourceID, CreatedBy: actor, CustomerID: b.CustomerID, IdempotencyKey: params.IdempotencyKey, RequestHash: params.RequestHash, StartsAt: params.StartsAt, EndsAt: params.EndsAt, BusyStartsAt: params.BusyStartsAt, BusyEndsAt: params.BusyEndsAt, SupersedesBookingID: id})
		if e != nil {
			return e
		}
		if e := q.CopyGuestContact(ctx, db.CopyGuestContactParams{Replacement: replacement.ID, Original: id}); e != nil {
			return e
		}
		out.Booking = customerBookingView(b)
		out.Booking.Status = "rescheduled"
		out.Booking.ReplacementID = replacement.ID
		out.Replacement = CustomerBooking{BusinessName: b.BusinessName, ServiceName: b.ServiceName, LocationName: b.LocationName, ID: replacement.ID, BusinessID: b.BusinessID, ServiceID: b.ServiceID, LocationID: b.LocationID, StartsAt: replacement.StartsAt.Time.UTC(), EndsAt: replacement.EndsAt.Time.UTC(), Status: "confirmed", SupersedesID: id}
		response, e := json.Marshal(out)
		if e != nil {
			return e
		}
		return q.CreateBookingChange(ctx, db.CreateBookingChangeParams{BusinessID: b.BusinessID, ActorID: actor, IdempotencyKey: storedKey, BookingID: id, Action: "reschedule", RequestHash: hash, Response: response})
	})
	if err != nil {
		return CustomerReschedule{}, bookingError(err)
	}
	return out, nil
}

func (s *BusinessService) CustomerRescheduleSlots(ctx context.Context, actor int64, id pgtype.UUID, date string) (SlotResult, error) {
	day, e := time.Parse("2006-01-02", date)
	if e != nil || len(date) != 10 || day.Year() < 2000 || day.Year() > 2100 || !id.Valid {
		return SlotResult{}, ErrInvalidBooking
	}
	var out SlotResult
	e = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		b, e := ownedBookingForChange(ctx, q, actor, id)
		if e != nil {
			return e
		}
		if e = customerMoveAllowed(ctx, q, b); e != nil {
			return e
		}
		in := SlotRequest{Date: date, LocationID: b.LocationID, StaffID: b.StaffID, ResourceID: b.ResourceID}
		out, e = calculateCatalogSlots(ctx, q, b.BusinessID, b.ServiceID, in, day, int32(b.EndsAt.Time.Sub(b.StartsAt.Time)/time.Minute))
		if e != nil {
			return e
		}
		out, e = filterBookingOccupancy(ctx, q, b.BusinessID, b.ServiceID, in, out, id)
		if e != nil {
			return e
		}
		slots := []Slot{}
		for _, slot := range out.Slots {
			if !slot.StartsAt.Equal(b.StartsAt.Time) {
				slots = append(slots, slot)
			}
		}
		out.Slots = slots
		return nil
	})
	if e != nil {
		return SlotResult{}, bookingError(e)
	}
	return out, nil
}
