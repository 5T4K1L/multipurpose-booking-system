package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type BookingCalendarStore interface {
	ListCalendarBookings(context.Context, db.ListCalendarBookingsParams) ([]db.ListCalendarBookingsRow, error)
	CalendarBookingCounts(context.Context, db.CalendarBookingCountsParams) (db.CalendarBookingCountsRow, error)
}
type CalendarEntry struct {
	BookingView
	ServiceName  string `json:"service_name"`
	LocationName string `json:"location_name"`
	StaffName    string `json:"staff_name"`
	ResourceName string `json:"resource_name"`
}
type CalendarCounts struct {
	Total       int64 `json:"total"`
	Confirmed   int64 `json:"confirmed"`
	Completed   int64 `json:"completed"`
	Cancelled   int64 `json:"cancelled"`
	Rescheduled int64 `json:"rescheduled"`
}
type BookingCalendar struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	Timezone   string          `json:"timezone"`
	Counts     CalendarCounts  `json:"counts"`
	Items      []CalendarEntry `json:"items"`
	NextCursor string          `json:"next_cursor"`
}

func (s *BookingService) Calendar(ctx context.Context, actor int64, business pgtype.UUID, from, to string, in PageRequest) (BookingCalendar, error) {
	start, e1 := time.Parse("2006-01-02", from)
	end, e2 := time.Parse("2006-01-02", to)
	if e1 != nil || e2 != nil || start.Year() < 2000 || end.Year() > 2100 || end.Before(start) || end.Sub(start) > 30*24*time.Hour {
		return BookingCalendar{}, ErrInvalidPage
	}
	return bookingRead(ctx, s, actor, business, func(q BusinessStore, a TenantAccess) (BookingCalendar, error) {
		zone := a.business.Timezone
		limit, c, err := parseCatalogPage(in, "Calendar:"+from+":"+to+":"+zone, actor, business, pgtype.UUID{})
		if err != nil {
			return BookingCalendar{}, err
		}
		after := pgtype.Timestamptz{Time: start, Valid: true}
		if in.Cursor != "" {
			row, e := q.GetBookingView(ctx, db.GetBookingViewParams{BusinessID: a.BusinessID(), ID: c.After})
			if errors.Is(e, pgx.ErrNoRows) {
				return BookingCalendar{}, ErrInvalidPage
			}
			if e != nil {
				return BookingCalendar{}, e
			}
			after = row.StartsAt
		}
		begin, finish := pgtype.Date{Time: start, Valid: true}, pgtype.Date{Time: end, Valid: true}
		counts, err := q.CalendarBookingCounts(ctx, db.CalendarBookingCountsParams{BusinessID: a.BusinessID(), Timezone: zone, FromDate: begin, ToDate: finish})
		if err != nil {
			return BookingCalendar{}, err
		}
		rows, err := q.ListCalendarBookings(ctx, db.ListCalendarBookingsParams{BusinessID: a.BusinessID(), Timezone: zone, FromDate: begin, ToDate: finish, HasCursor: in.Cursor != "", AfterStart: after, AfterID: c.After, PageSize: int32(limit + 1)})
		if err != nil {
			return BookingCalendar{}, err
		}
		entries := []CalendarEntry{}
		for _, v := range rows {
			view := BookingView{ID: v.ID, BookingInput: BookingInput{v.ServiceID, v.LocationID, v.StaffID, v.ResourceID, v.StartsAt.Time.UTC()}, EndsAt: v.EndsAt.Time.UTC(), BusyStartsAt: v.BusyStartsAt.Time.UTC(), BusyEndsAt: v.BusyEndsAt.Time.UTC(), Status: v.Status, SupersedesID: v.SupersedesBookingID, ReplacementID: v.ReplacementID}
			entries = append(entries, CalendarEntry{view, v.ServiceName, v.LocationName, v.StaffName, v.ResourceName})
		}
		page := finishCatalogPage(entries, limit, c, func(v CalendarEntry) pgtype.UUID { return v.ID })
		return BookingCalendar{from, to, zone, CalendarCounts{counts.Total, counts.Confirmed, counts.Completed, counts.Cancelled, counts.Rescheduled}, page.Items, page.NextCursor}, nil
	})
}
