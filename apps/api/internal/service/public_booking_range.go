package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

// The whole interval must have the SAME eligible allocation, including buffers.
func (s *BusinessService) PublicBookingRange(ctx context.Context, business, serviceID, location pgtype.UUID, start, end time.Time) (SlotResult, error) {
	if !business.Valid || !serviceID.Valid || !location.Valid || start.Year() < 2000 || end.Year() > 2100 || !end.After(start) || end.Sub(start) > 7*24*time.Hour || start.Second() != 0 || end.Second() != 0 || start.Nanosecond() != 0 || end.Nanosecond() != 0 {
		return SlotResult{}, ErrInvalidCatalog
	}
	var out SlotResult
	e := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if _, e := q.LockBusiness(ctx, business); e != nil {
			return e
		}
		loc, e := publishedSelection(ctx, q, business, serviceID, location)
		if e != nil {
			return e
		}
		duration, e := customerDuration(ctx, q, business, serviceID, int32(end.Sub(start)/time.Minute))
		if e != nil {
			return e
		}
		zone, e := time.LoadLocation(loc.Timezone)
		if e != nil {
			return e
		}
		date := start.In(zone).Format("2006-01-02")
		day, _ := time.Parse("2006-01-02", date)
		choices, e := q.PublicAvailabilityCandidates(ctx, db.PublicAvailabilityCandidatesParams{BusinessID: business, ServiceID: serviceID, LocationID: location})
		if e != nil {
			return e
		}
		if len(choices) > 32 {
			return ErrAvailabilityComplex
		}
		result, _, e := publicDayAvailability(ctx, q, business, serviceID, location, day, loc.Timezone, choices, duration)
		if e != nil {
			return e
		}
		out = SlotResult{Date: date, Timezone: loc.Timezone, Advisory: true, Slots: []Slot{}}
		for _, slot := range result.Slots {
			if slot.StartsAt.Equal(start) && slot.EndsAt.Equal(end) {
				out.Slots = append(out.Slots, slot)
			}
		}
		return nil
	})
	if errors.Is(e, pgx.ErrNoRows) {
		return SlotResult{}, ErrPublicProfileNotFound
	}
	if errors.Is(e, ErrInvalidBooking) {
		return SlotResult{}, ErrInvalidCatalog
	}
	return out, e
}
