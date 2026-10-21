package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

// Ownership and allocations are server-controlled. Expected end/timezone only
// detect stale selections; the saved duration always comes from current rules.
type CustomerBookingInput struct {
	Contact          *GuestContact `json:"contact,omitempty"`
	BusinessID       pgtype.UUID   `json:"business_id"`
	ServiceID        pgtype.UUID   `json:"service_id"`
	LocationID       pgtype.UUID   `json:"location_id"`
	StartsAt         time.Time     `json:"starts_at"`
	ExpectedEndsAt   time.Time     `json:"expected_ends_at"`
	ExpectedTimezone string        `json:"expected_timezone"`
}
type CustomerBookingCreated struct {
	CustomerBooking
	Replayed bool `json:"replayed"`
}

func (s *BusinessService) CreateCustomerBooking(ctx context.Context, actor int64, key string, in CustomerBookingInput) (CustomerBookingCreated, error) {
	if !validBookingKey(key) || !in.BusinessID.Valid || !in.ServiceID.Valid || !in.LocationID.Valid || in.StartsAt.Year() < 2000 || in.StartsAt.Year() > 2100 || in.StartsAt.Second() != 0 || in.StartsAt.Nanosecond() != 0 || !in.ExpectedEndsAt.After(in.StartsAt) || in.ExpectedEndsAt.Sub(in.StartsAt) > 10080*time.Minute || in.ExpectedEndsAt.Second() != 0 || in.ExpectedEndsAt.Nanosecond() != 0 || in.ExpectedTimezone == "" || len(in.ExpectedTimezone) > 100 {
		return CustomerBookingCreated{}, ErrInvalidBooking
	}
	if in.Contact != nil {
		contact, e := cleanGuestContact(*in.Contact)
		if e != nil {
			return CustomerBookingCreated{}, e
		}
		in.Contact = &contact
	}
	in.StartsAt = in.StartsAt.UTC()
	in.ExpectedEndsAt = in.ExpectedEndsAt.UTC()
	payload, _ := json.Marshal(in)
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	// Reserved namespace separates customer and operator requests, including for
	// users who also own a business. Hashing preserves the DB key length bound.
	keyDigest := sha256.Sum256([]byte(key))
	storedKey := "~customer:" + hex.EncodeToString(keyDigest[:])
	var out CustomerBookingCreated
	err := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if _, e := q.LockBusiness(ctx, in.BusinessID); e != nil {
			return e
		}
		if e := activeBusinessUser(ctx, q, actor); e != nil {
			return e
		}
		existing, e := q.GetBookingReplay(ctx, db.GetBookingReplayParams{BusinessID: in.BusinessID, CreatedBy: actor, IdempotencyKey: storedKey})
		if e == nil {
			if existing.RequestHash != hash || !existing.CustomerID.Valid || existing.CustomerID.Int64 != actor {
				return ErrIdempotencyConflict
			}
			b, e := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: existing.ID, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
			if e != nil {
				return e
			}
			out = CustomerBookingCreated{CustomerBooking: CustomerBooking{b.BusinessName, b.ServiceName, b.LocationName, b.ID, b.BusinessID, b.ServiceID, b.LocationID, b.StartsAt.Time.UTC(), b.EndsAt.Time.UTC(), b.Status, b.SupersedesBookingID, b.ReplacementID}, Replayed: true}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		loc, e := publishedSelection(ctx, q, in.BusinessID, in.ServiceID, in.LocationID)
		if e != nil {
			return e
		}
		duration, e := customerDuration(ctx, q, in.BusinessID, in.ServiceID, int32(in.ExpectedEndsAt.Sub(in.StartsAt)/time.Minute))
		if e != nil {
			return e
		}
		if loc.Timezone != in.ExpectedTimezone {
			return ErrPublicationChanged
		}
		zone, e := time.LoadLocation(loc.Timezone)
		if e != nil {
			return e
		}
		date := in.StartsAt.In(zone).Format("2006-01-02")
		day, e := time.Parse("2006-01-02", date)
		if e != nil || day.Year() < 2000 || day.Year() > 2100 {
			return ErrInvalidBooking
		}
		choices, e := q.PublicAvailabilityCandidates(ctx, db.PublicAvailabilityCandidatesParams{BusinessID: in.BusinessID, ServiceID: in.ServiceID, LocationID: in.LocationID})
		if e != nil {
			return e
		}
		if len(choices) > 32 {
			return ErrAvailabilityComplex
		}
		for _, choice := range choices {
			request := SlotRequest{Date: date, LocationID: in.LocationID, StaffID: choice.StaffID, ResourceID: choice.ResourceID}
			result, e := calculateCatalogSlots(ctx, q, in.BusinessID, in.ServiceID, request, day, duration)
			if e != nil {
				return e
			}
			result, e = filterBookingOccupancy(ctx, q, in.BusinessID, in.ServiceID, request, result)
			if e != nil {
				return e
			}
			for _, slot := range result.Slots {
				if !slot.StartsAt.Equal(in.StartsAt) {
					continue
				}
				if !slot.EndsAt.Equal(in.ExpectedEndsAt) {
					return ErrPublicationChanged
				}
				rules, e := effectiveSchedulingRules(ctx, q, db.GetServiceSchedulingRulesParams{BusinessID: in.BusinessID, ServiceID: in.ServiceID})
				if e != nil {
					return e
				}
				stamp := func(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
				b, e := q.CreateCustomerBooking(ctx, db.CreateCustomerBookingParams{BusinessID: in.BusinessID, ServiceID: in.ServiceID, LocationID: in.LocationID, StaffID: choice.StaffID, ResourceID: choice.ResourceID, CreatedBy: actor, IdempotencyKey: storedKey, RequestHash: hash, StartsAt: stamp(slot.StartsAt), EndsAt: stamp(slot.EndsAt), BusyStartsAt: stamp(slot.StartsAt.Add(-time.Duration(rules.BufferBeforeMinutes) * time.Minute)), BusyEndsAt: stamp(slot.EndsAt.Add(time.Duration(rules.BufferAfterMinutes) * time.Minute))})
				if e != nil {
					return e
				}
				if in.Contact != nil {
					c := in.Contact
					if e := q.PutGuestContact(ctx, db.PutGuestContactParams{BookingID: b.ID, Name: c.Name, Phone: c.Phone, Email: c.Email, Notes: c.Notes}); e != nil {
						return e
					}
				}
				view, e := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: b.ID, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
				if e != nil {
					return e
				}
				out = CustomerBookingCreated{CustomerBooking: customerBookingView(view)}
				return nil
			}
		}
		return ErrBookingConflict
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerBookingCreated{}, ErrPublicProfileNotFound
	}
	if err != nil {
		if errors.Is(err, ErrPublicProfileNotFound) || errors.Is(err, ErrPublicationChanged) || errors.Is(err, ErrAvailabilityComplex) {
			return CustomerBookingCreated{}, err
		}
		return CustomerBookingCreated{}, bookingError(err)
	}
	return out, nil
}
