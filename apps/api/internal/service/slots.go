package service

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type SlotRequest struct {
	Date                            string
	LocationID, StaffID, ResourceID pgtype.UUID
}
type Slot struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}
type SlotResult struct {
	Date     string `json:"date"`
	Timezone string `json:"timezone"`
	Advisory bool   `json:"advisory"`
	Slots    []Slot `json:"slots"`
}

func (s *BookingCatalogService) CalculateSlots(ctx context.Context, actor int64, business, id pgtype.UUID, in SlotRequest) (SlotResult, error) {
	day, e := time.Parse("2006-01-02", in.Date)
	if e != nil || len(in.Date) != 10 || day.Year() < 2000 || day.Year() > 2100 || !id.Valid || !in.LocationID.Valid {
		return SlotResult{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (SlotResult, error) {
		result, e := calculateCatalogSlots(ctx, q, a.BusinessID(), id, in, day)
		if e != nil {
			return SlotResult{}, e
		}
		return filterBookingOccupancy(ctx, q, a.BusinessID(), id, in, result)
	})
}

// Shared by advisory reads and booking creation inside the caller's transaction.
func calculateCatalogSlots(ctx context.Context, q BookingCatalogStore, business, id pgtype.UUID, in SlotRequest, day time.Time, requestedDuration ...int32) (SlotResult, error) {
	offered, e := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: id})
	if e != nil {
		return SlotResult{}, e
	}
	loc, e := q.GetLocation(ctx, db.GetLocationParams{BusinessID: business, ID: in.LocationID})
	if e != nil {
		return SlotResult{}, e
	}
	zone, e := time.LoadLocation(loc.Timezone)
	if e != nil {
		return SlotResult{}, e
	}
	result := SlotResult{Date: in.Date, Timezone: loc.Timezone, Advisory: true, Slots: []Slot{}}
	// Business location hours govern availability; staff and resources follow them.
	if in.StaffID.Valid {
		if e := availabilityEntity(ctx, q, business, in.StaffID, "staff"); e != nil {
			return SlotResult{}, e
		}
	}
	if in.ResourceID.Valid {
		r, e := q.GetResource(ctx, db.GetResourceParams{BusinessID: business, ID: in.ResourceID})
		if e != nil {
			return SlotResult{}, e
		}
		if r.LocationID != in.LocationID {
			return result, nil
		}
	}
	eligible, e := q.CheckSlotEligibility(ctx, db.CheckSlotEligibilityParams{BusinessID: business, ServiceID: id, LocationID: in.LocationID, WithStaff: in.StaffID.Valid, StaffID: in.StaffID, WithResource: in.ResourceID.Valid, ResourceID: in.ResourceID})
	if e != nil {
		return SlotResult{}, e
	}
	if !eligible.ServiceLocation || !eligible.StaffEligible.Valid || !eligible.StaffEligible.Bool || !eligible.ResourceEligible.Valid || !eligible.ResourceEligible.Bool {
		return result, nil
	}
	rules, e := effectiveSchedulingRules(ctx, q, db.GetServiceSchedulingRulesParams{BusinessID: business, ServiceID: id})
	if e != nil {
		return SlotResult{}, e
	}
	hours, e := q.ListLocationWeeklyHours(ctx, db.ListLocationWeeklyHoursParams{BusinessID: business, LocationID: in.LocationID})
	if e != nil {
		return SlotResult{}, e
	}
	dates, e := q.ListLocationClosedDates(ctx, db.ListLocationClosedDatesParams{BusinessID: business, LocationID: in.LocationID})
	if e != nil {
		return SlotResult{}, e
	}
	weekly := []WeeklyHours{}
	for _, h := range hours {
		weekly = append(weekly, WeeklyHours{h.Weekday, h.StartMinute, h.EndMinute})
	}
	closed := map[string]bool{}
	for _, d := range dates {
		closed[d.Time.Format("2006-01-02")] = true
	}
	if offered.AllDay {
		result.Slots, e = wholeBusinessDay(ctx, day, zone, weekly, closed, SchedulingRulesInput{rules.BufferBeforeMinutes, rules.BufferAfterMinutes, rules.MinimumLeadMinutes}, time.Now().UTC())
		return result, e
	}
	duration := offered.DurationMinutes
	if len(requestedDuration) > 0 {
		duration = requestedDuration[0]
	}
	result.Slots, e = calculateSlots(ctx, day, zone, weekly, closed, duration, SchedulingRulesInput{rules.BufferBeforeMinutes, rules.BufferAfterMinutes, rules.MinimumLeadMinutes}, nil, time.Now().UTC(), offered.AllDay)
	if e != nil {
		return SlotResult{}, e
	}
	return result, nil
}
func windowCovers(windows []AvailabilityWindow, start, end time.Time) bool {
	for _, w := range windows {
		if w.EndsAt.Compare(start) <= 0 {
			continue
		}
		if w.StartsAt.After(start) {
			return false
		}
		start = w.EndsAt
		if !start.Before(end) {
			return true
		}
	}
	return false
}

// Walk real instants rather than constructing local times (which can be missing
// or repeated). Prefix sums verify every elapsed minute of buffered occupancy.
// Whole-day services require an open start, then may continue overnight. Explicit
// closed dates still block the entire occupied interval, as do allocations.
// The fixed one-date/15-minute grid bounds work and returns both fall-back folds.
func calculateSlots(ctx context.Context, day time.Time, zone *time.Location, hours []WeeklyHours, closed map[string]bool, duration int32, rules SchedulingRulesInput, windows [][]AvailabilityWindow, now time.Time, overnight ...bool) ([]Slot, error) {
	out := []Slot{}
	if duration < 1 || duration > 10080 || rules.BufferBeforeMinutes < 0 || rules.BufferBeforeMinutes > 1440 || rules.BufferAfterMinutes < 0 || rules.BufferAfterMinutes > 1440 || rules.MinimumLeadMinutes < 0 || rules.MinimumLeadMinutes > 525600 {
		return nil, ErrInvalidCatalog
	}
	continueOvernight := len(overnight) > 0 && overnight[0]
	before, after, length := int(rules.BufferBeforeMinutes), int(rules.BufferAfterMinutes), int(duration)
	start := day.Add(-48 * time.Hour)
	n := 4*1440 + before + after + length
	blocked := make([]int, n+1)
	for i := 0; i < n; i++ {
		if i%256 == 0 {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
		}
		local := start.Add(time.Duration(i) * time.Minute).In(zone)
		minute := int32(local.Hour()*60 + local.Minute())
		open := false
		if !closed[local.Format("2006-01-02")] {
			if continueOvernight {
				open = true
			}
			for _, h := range hours {
				if h.Weekday == int32(local.Weekday()) && h.StartMinute <= minute && minute < h.EndMinute {
					open = true
					break
				}
			}
		}
		blocked[i+1] = blocked[i]
		if !open {
			blocked[i+1]++
		}
	}
	earliest := now.Add(time.Duration(rules.MinimumLeadMinutes) * time.Minute)
	for i := before; i+length+after <= n; i++ {
		if i%256 == 0 {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
		}
		instant := start.Add(time.Duration(i) * time.Minute)
		local := instant.In(zone)
		if local.Format("2006-01-02") != day.Format("2006-01-02") || local.Minute()%15 != 0 || local.Second() != 0 || instant.Before(earliest) {
			continue
		}
		if continueOvernight {
			startsDuringHours := false
			minute := int32(local.Hour()*60 + local.Minute())
			for _, h := range hours {
				if h.Weekday == int32(local.Weekday()) && h.StartMinute <= minute && minute < h.EndMinute {
					startsDuringHours = true
					break
				}
			}
			if !startsDuringHours {
				continue
			}
		}
		if blocked[i+length+after] != blocked[i-before] {
			continue
		}
		end := instant.Add(time.Duration(length) * time.Minute)
		busyStart, busyEnd := instant.Add(-time.Duration(before)*time.Minute), end.Add(time.Duration(after)*time.Minute)
		if busyStart.Before(now) {
			continue
		}
		allowed := true
		for _, w := range windows {
			if !windowCovers(w, busyStart, busyEnd) {
				allowed = false
				break
			}
		}
		if allowed {
			out = append(out, Slot{StartsAt: instant.UTC(), EndsAt: end.UTC()})
		}
	}
	return out, nil
}

// Reserve opening through closing on one local date, including any midday break.
func wholeBusinessDay(ctx context.Context, day time.Time, zone *time.Location, hours []WeeklyHours, closed map[string]bool, rules SchedulingRulesInput, now time.Time) ([]Slot, error) {
	out := []Slot{}
	date := day.Format("2006-01-02")
	if closed[date] {
		return out, nil
	}
	var first, last time.Time
	for instant, end := day.Add(-48*time.Hour), day.Add(48*time.Hour); instant.Before(end); instant = instant.Add(time.Minute) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		local := instant.In(zone)
		if local.Format("2006-01-02") != date {
			continue
		}
		minute := int32(local.Hour()*60 + local.Minute())
		for _, h := range hours {
			if int32(local.Weekday()) == h.Weekday && minute >= h.StartMinute && minute < h.EndMinute {
				if first.IsZero() {
					first = instant
				}
				last = instant.Add(time.Minute)
				break
			}
		}
	}
	if first.IsZero() || first.Before(now.Add(time.Duration(rules.MinimumLeadMinutes)*time.Minute)) {
		return out, nil
	}
	busyStart, busyEnd := first.Add(-time.Duration(rules.BufferBeforeMinutes)*time.Minute), last.Add(time.Duration(rules.BufferAfterMinutes)*time.Minute)
	if busyStart.Before(now) {
		return out, nil
	}
	for t := busyStart; t.Before(busyEnd); t = t.Add(time.Minute) {
		if closed[t.In(zone).Format("2006-01-02")] {
			return out, nil
		}
	}
	return append(out, Slot{StartsAt: first, EndsAt: last}), nil
}
