package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

type SchedulingStore interface {
	ListLocationWeeklyHours(context.Context, db.ListLocationWeeklyHoursParams) ([]db.ListLocationWeeklyHoursRow, error)
	DeleteLocationWeeklyHours(context.Context, db.DeleteLocationWeeklyHoursParams) error
	CreateLocationWeeklyHours(context.Context, db.CreateLocationWeeklyHoursParams) error
	ListLocationClosedDates(context.Context, db.ListLocationClosedDatesParams) ([]pgtype.Date, error)
	DeleteLocationClosedDates(context.Context, db.DeleteLocationClosedDatesParams) error
	CreateLocationClosedDate(context.Context, db.CreateLocationClosedDateParams) error
	GetServiceSchedulingRules(context.Context, db.GetServiceSchedulingRulesParams) (db.ServiceSchedulingRule, error)
	PutServiceSchedulingRules(context.Context, db.PutServiceSchedulingRulesParams) error
}
type WeeklyHours struct {
	Weekday     int32 `json:"weekday"`
	StartMinute int32 `json:"start_minute"`
	EndMinute   int32 `json:"end_minute"`
}
type LocationScheduleInput struct {
	WeeklyHours []WeeklyHours `json:"weekly_hours"`
	ClosedDates []string      `json:"closed_dates"`
}
type LocationSchedule struct {
	Timezone    string        `json:"timezone"`
	WeeklyHours []WeeklyHours `json:"weekly_hours"`
	ClosedDates []string      `json:"closed_dates"`
}
type SchedulingRulesInput struct {
	BufferBeforeMinutes int32 `json:"buffer_before_minutes"`
	BufferAfterMinutes  int32 `json:"buffer_after_minutes"`
	MinimumLeadMinutes  int32 `json:"minimum_lead_minutes"`
}
type SchedulingRules struct {
	Configured bool `json:"configured"`
	SchedulingRulesInput
}

// Weekly intervals are half-open local wall-clock minutes. Overnight intervals
// must be split at midnight; 1440 is permitted only as an end boundary.
func normalizeSchedule(in LocationScheduleInput) (LocationScheduleInput, error) {
	if in.WeeklyHours == nil || in.ClosedDates == nil || len(in.WeeklyHours) > 28 || len(in.ClosedDates) > 128 {
		return LocationScheduleInput{}, ErrInvalidCatalog
	}
	out := LocationScheduleInput{WeeklyHours: append([]WeeklyHours{}, in.WeeklyHours...), ClosedDates: append([]string{}, in.ClosedDates...)}
	sort.Slice(out.WeeklyHours, func(i, j int) bool {
		a, b := out.WeeklyHours[i], out.WeeklyHours[j]
		if a.Weekday != b.Weekday {
			return a.Weekday < b.Weekday
		}
		return a.StartMinute < b.StartMinute
	})
	for i, h := range out.WeeklyHours {
		if h.Weekday < 0 || h.Weekday > 6 || h.StartMinute < 0 || h.EndMinute > 1440 || h.StartMinute >= h.EndMinute {
			return LocationScheduleInput{}, ErrInvalidCatalog
		}
		if i > 0 {
			prev := out.WeeklyHours[i-1]
			if prev.Weekday == h.Weekday && prev.EndMinute > h.StartMinute {
				return LocationScheduleInput{}, ErrInvalidCatalog
			}
		}
	}
	sort.Strings(out.ClosedDates)
	for i, d := range out.ClosedDates {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil || len(d) != 10 || parsed.Year() < 1 || parsed.Format("2006-01-02") != d || (i > 0 && out.ClosedDates[i-1] == d) {
			return LocationScheduleInput{}, ErrInvalidCatalog
		}
	}
	return out, nil
}
func (s *BookingCatalogService) PutLocationSchedule(ctx context.Context, actor int64, business, location pgtype.UUID, in LocationScheduleInput) (LocationSchedule, error) {
	normalized, err := normalizeSchedule(in)
	if err != nil || !location.Valid {
		return LocationSchedule{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (LocationSchedule, error) {
		loc, err := q.GetLocation(ctx, db.GetLocationParams{BusinessID: a.BusinessID(), ID: location})
		if err != nil {
			return LocationSchedule{}, err
		}
		if err = q.DeleteLocationWeeklyHours(ctx, db.DeleteLocationWeeklyHoursParams{BusinessID: a.BusinessID(), LocationID: location}); err != nil {
			return LocationSchedule{}, err
		}
		if err = q.DeleteLocationClosedDates(ctx, db.DeleteLocationClosedDatesParams{BusinessID: a.BusinessID(), LocationID: location}); err != nil {
			return LocationSchedule{}, err
		}
		for _, h := range normalized.WeeklyHours {
			if err = q.CreateLocationWeeklyHours(ctx, db.CreateLocationWeeklyHoursParams{BusinessID: a.BusinessID(), LocationID: location, Weekday: h.Weekday, StartMinute: h.StartMinute, EndMinute: h.EndMinute}); err != nil {
				return LocationSchedule{}, err
			}
		}
		for _, d := range normalized.ClosedDates {
			date, _ := time.Parse("2006-01-02", d)
			if err = q.CreateLocationClosedDate(ctx, db.CreateLocationClosedDateParams{BusinessID: a.BusinessID(), LocationID: location, ClosedDate: pgtype.Date{Time: date, Valid: true}}); err != nil {
				return LocationSchedule{}, err
			}
		}
		return LocationSchedule{Timezone: loc.Timezone, WeeklyHours: normalized.WeeklyHours, ClosedDates: normalized.ClosedDates}, nil
	})
}
func (s *BookingCatalogService) GetLocationSchedule(ctx context.Context, actor int64, business, location pgtype.UUID) (LocationSchedule, error) {
	if !location.Valid {
		return LocationSchedule{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (LocationSchedule, error) {
		loc, err := q.GetLocation(ctx, db.GetLocationParams{BusinessID: a.BusinessID(), ID: location})
		if err != nil {
			return LocationSchedule{}, err
		}
		rows, err := q.ListLocationWeeklyHours(ctx, db.ListLocationWeeklyHoursParams{BusinessID: a.BusinessID(), LocationID: location})
		if err != nil {
			return LocationSchedule{}, err
		}
		dates, err := q.ListLocationClosedDates(ctx, db.ListLocationClosedDatesParams{BusinessID: a.BusinessID(), LocationID: location})
		if err != nil {
			return LocationSchedule{}, err
		}
		out := LocationSchedule{Timezone: loc.Timezone, WeeklyHours: []WeeklyHours{}, ClosedDates: []string{}}
		for _, h := range rows {
			out.WeeklyHours = append(out.WeeklyHours, WeeklyHours{h.Weekday, h.StartMinute, h.EndMinute})
		}
		for _, d := range dates {
			out.ClosedDates = append(out.ClosedDates, d.Time.Format("2006-01-02"))
		}
		return out, nil
	})
}
func (s *BookingCatalogService) PutSchedulingRules(ctx context.Context, actor int64, business, id pgtype.UUID, in SchedulingRulesInput) (SchedulingRules, error) {
	if !id.Valid || in.BufferBeforeMinutes < 0 || in.BufferBeforeMinutes > 1440 || in.BufferAfterMinutes < 0 || in.BufferAfterMinutes > 1440 || in.MinimumLeadMinutes < 0 || in.MinimumLeadMinutes > 525600 {
		return SchedulingRules{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (SchedulingRules, error) {
		if _, err := q.GetService(ctx, db.GetServiceParams{BusinessID: a.BusinessID(), ID: id}); err != nil {
			return SchedulingRules{}, err
		}
		if err := q.PutServiceSchedulingRules(ctx, db.PutServiceSchedulingRulesParams{BusinessID: a.BusinessID(), ServiceID: id, BufferBeforeMinutes: in.BufferBeforeMinutes, BufferAfterMinutes: in.BufferAfterMinutes, MinimumLeadMinutes: in.MinimumLeadMinutes}); err != nil {
			return SchedulingRules{}, err
		}
		return SchedulingRules{Configured: true, SchedulingRulesInput: in}, nil
	})
}
func (s *BookingCatalogService) GetSchedulingRules(ctx context.Context, actor int64, business, id pgtype.UUID) (SchedulingRules, error) {
	if !id.Valid {
		return SchedulingRules{}, ErrInvalidCatalog
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (SchedulingRules, error) {
		if _, err := q.GetService(ctx, db.GetServiceParams{BusinessID: a.BusinessID(), ID: id}); err != nil {
			return SchedulingRules{}, err
		}
		row, err := q.GetServiceSchedulingRules(ctx, db.GetServiceSchedulingRulesParams{BusinessID: a.BusinessID(), ServiceID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return SchedulingRules{}, nil
		}
		if err != nil {
			return SchedulingRules{}, err
		}
		return SchedulingRules{Configured: true, SchedulingRulesInput: SchedulingRulesInput{row.BufferBeforeMinutes, row.BufferAfterMinutes, row.MinimumLeadMinutes}}, nil
	})
}

// Preparation, cleanup and notice are optional. Missing customization uses zero
// minutes; database errors must never be mistaken for an available schedule.
func effectiveSchedulingRules(ctx context.Context, q SchedulingStore, in db.GetServiceSchedulingRulesParams) (db.ServiceSchedulingRule, error) {
	row, err := q.GetServiceSchedulingRules(ctx, in)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ServiceSchedulingRule{BusinessID: in.BusinessID, ServiceID: in.ServiceID}, nil
	}
	return row, err
}
