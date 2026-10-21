package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"sort"
	"time"
)

type PublicAvailabilityStore interface {
	ListPublicationServiceLocations(context.Context, pgtype.UUID) ([]db.ListPublicationServiceLocationsRow, error)
	PublicAvailabilityCandidates(context.Context, db.PublicAvailabilityCandidatesParams) ([]db.PublicAvailabilityCandidatesRow, error)
}

var ErrAvailabilityComplex = errors.New("availability configuration exceeds request limit")

// Reads are serialized with tenant writers. No customer/membership identity is
// fabricated: the public snapshot itself authorizes this advisory read only.
func (s *BusinessService) PublicAvailability(ctx context.Context, business, serviceID, location pgtype.UUID, date string) (SlotResult, error) {
	day, err := time.Parse("2006-01-02", date)
	if err != nil || len(date) != 10 || day.Year() < 2000 || day.Year() > 2100 || !business.Valid || !serviceID.Valid || !location.Valid {
		return SlotResult{}, ErrInvalidCatalog
	}
	var out SlotResult
	err = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if _, e := q.LockBusiness(ctx, business); e != nil {
			return e
		}
		loc, e := publishedSelection(ctx, q, business, serviceID, location)
		if e != nil {
			return e
		}
		out = SlotResult{Date: date, Timezone: loc.Timezone, Advisory: true, Slots: []Slot{}}
		choices, e := q.PublicAvailabilityCandidates(ctx, db.PublicAvailabilityCandidatesParams{BusinessID: business, ServiceID: serviceID, LocationID: location})
		if e != nil {
			return e
		}
		if len(choices) > 32 {
			return ErrAvailabilityComplex
		}
		out, _, e = publicDayAvailability(ctx, q, business, serviceID, location, day, loc.Timezone, choices)
		return e
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SlotResult{}, ErrPublicProfileNotFound
	}
	if err != nil {
		return SlotResult{}, err
	}
	return out, nil
}

// Calendar days describe remaining times, never booking/customer identities.
type PublicCalendarDay struct {
	Date   string `json:"date"`
	Status string `json:"status"`
	Slots  []Slot `json:"slots"`
}
type PublicCalendar struct {
	Month    string              `json:"month"`
	Timezone string              `json:"timezone"`
	Advisory bool                `json:"advisory"`
	Days     []PublicCalendarDay `json:"days"`
}

// A single bounded month and one business lock give the calendar a consistent
// snapshot. The same scheduling/occupancy rules also validate booking creation.
func (s *BusinessService) PublicAvailabilityCalendar(ctx context.Context, business, serviceID, location pgtype.UUID, month string) (PublicCalendar, error) {
	first, err := time.Parse("2006-01", month)
	if err != nil || len(month) != 7 || first.Year() < 2000 || first.Year() > 2100 || !business.Valid || !serviceID.Valid || !location.Valid {
		return PublicCalendar{}, ErrInvalidCatalog
	}
	var out PublicCalendar
	err = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if _, e := q.LockBusiness(ctx, business); e != nil {
			return e
		}
		loc, e := publishedSelection(ctx, q, business, serviceID, location)
		if e != nil {
			return e
		}
		choices, e := q.PublicAvailabilityCandidates(ctx, db.PublicAvailabilityCandidatesParams{BusinessID: business, ServiceID: serviceID, LocationID: location})
		if e != nil {
			return e
		}
		if len(choices) > 32 {
			return ErrAvailabilityComplex
		}
		out = PublicCalendar{Month: month, Timezone: loc.Timezone, Advisory: true, Days: []PublicCalendarDay{}}
		for day := first; day.Month() == first.Month(); day = day.AddDate(0, 0, 1) {
			if e := ctx.Err(); e != nil {
				return e
			}
			result, scheduled, e := publicDayAvailability(ctx, q, business, serviceID, location, day, loc.Timezone, choices)
			if e != nil {
				return e
			}
			status := "unavailable"
			if len(result.Slots) > 0 {
				status = "available"
			} else if scheduled {
				status = "fully_booked"
			}
			out.Days = append(out.Days, PublicCalendarDay{Date: result.Date, Status: status, Slots: result.Slots})
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicCalendar{}, ErrPublicProfileNotFound
	}
	if err != nil {
		return PublicCalendar{}, err
	}
	return out, nil
}

// scheduled means at least one time satisfies current hours, lead time, windows
// and buffers BEFORE booking occupancy is applied. A closed/past day is not full.
func publicDayAvailability(ctx context.Context, q BusinessStore, business, serviceID, location pgtype.UUID, day time.Time, timezone string, choices []db.PublicAvailabilityCandidatesRow, requestedDuration ...int32) (SlotResult, bool, error) {
	out := SlotResult{Date: day.Format("2006-01-02"), Timezone: timezone, Advisory: true, Slots: []Slot{}}
	scheduled := false
	seen := map[int64]bool{}
	if len(requestedDuration) == 0 {
		options, e := q.GetServiceBookingOptions(ctx, db.GetServiceBookingOptionsParams{BusinessID: business, ID: serviceID})
		if e != nil {
			return SlotResult{}, false, e
		}
		if options.Enabled {
			requestedDuration = []int32{options.MinimumMinutes}
		}
	}
	for _, choice := range choices {
		request := SlotRequest{Date: out.Date, LocationID: location, StaffID: choice.StaffID, ResourceID: choice.ResourceID}
		result, e := calculateCatalogSlots(ctx, q, business, serviceID, request, day, requestedDuration...)
		if e != nil {
			return SlotResult{}, false, e
		}
		scheduled = scheduled || len(result.Slots) > 0
		result, e = filterBookingOccupancy(ctx, q, business, serviceID, request, result)
		if e != nil {
			return SlotResult{}, false, e
		}
		for _, slot := range result.Slots {
			if !seen[slot.StartsAt.UnixNano()] {
				seen[slot.StartsAt.UnixNano()] = true
				out.Slots = append(out.Slots, slot)
			}
		}
	}
	sort.Slice(out.Slots, func(i, j int) bool { return out.Slots[i].StartsAt.Before(out.Slots[j].StartsAt) })
	return out, scheduled, nil
}

// The caller holds LockBusiness for the entire read or booking transaction.
func publishedSelection(ctx context.Context, q BusinessStore, business, serviceID, location pgtype.UUID) (db.Location, error) {
	data, e := q.GetPublicBusinessProfile(ctx, business)
	if e != nil {
		return db.Location{}, e
	}
	var profile PublicBusinessProfile
	if e = json.Unmarshal(data, &profile); e != nil {
		return db.Location{}, e
	}
	var selected *PublicService
	var place *PublicLocation
	for i := range profile.Services {
		if profile.Services[i].ID == serviceID {
			selected = &profile.Services[i]
			break
		}
	}
	if selected == nil {
		return db.Location{}, ErrPublicProfileNotFound
	}
	for i := range selected.Locations {
		if selected.Locations[i].ID == location {
			place = &selected.Locations[i]
			break
		}
	}
	if place == nil {
		return db.Location{}, ErrPublicProfileNotFound
	}
	offered, e := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: serviceID})
	if e != nil {
		return db.Location{}, e
	}
	loc, e := q.GetLocation(ctx, db.GetLocationParams{BusinessID: business, ID: location})
	if e != nil {
		return db.Location{}, e
	}
	if offered.AllDay != selected.AllDay || offered.DurationMinutes != selected.DurationMinutes || loc.Timezone != place.Timezone {
		return db.Location{}, ErrPublicationChanged
	}
	options, e := q.GetServiceBookingOptions(ctx, db.GetServiceBookingOptionsParams{BusinessID: business, ID: serviceID})
	if e != nil {
		return db.Location{}, e
	}
	if options.Enabled {
		if selected.BookingOptions == nil || *selected.BookingOptions != bookingOptions(options) {
			return db.Location{}, ErrPublicationChanged
		}
	} else if selected.BookingOptions != nil && selected.BookingOptions.Enabled {
		return db.Location{}, ErrPublicationChanged
	}
	eligible, e := q.CheckSlotEligibility(ctx, db.CheckSlotEligibilityParams{BusinessID: business, ServiceID: serviceID, LocationID: location})
	if e != nil {
		return db.Location{}, e
	}
	if !eligible.ServiceLocation {
		return db.Location{}, ErrPublicProfileNotFound
	}
	return loc, nil
}
