package service

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"reflect"
	"testing"
)

func TestNormalizeLocationSchedule(t *testing.T) {
	input := LocationScheduleInput{WeeklyHours: []WeeklyHours{{1, 780, 1020}, {0, 0, 1440}, {1, 540, 720}, {1, 720, 780}}, ClosedDates: []string{"2028-02-29", "2026-03-08"}}
	out, err := normalizeSchedule(input)
	if err != nil || out.WeeklyHours[0].Weekday != 0 || out.ClosedDates[0] != "2026-03-08" {
		t.Fatal(out, err)
	}
	if input.WeeklyHours[0].StartMinute != 780 || input.ClosedDates[0] != "2028-02-29" {
		t.Fatal("caller data mutated")
	}
	for _, hours := range [][]WeeklyHours{{{0, 0, 0}}, {{1, 720, 540}}, {{1, 0, 1441}}, {{-1, 0, 60}}, {{7, 0, 60}}, {{1, -1, 60}}, {{1, 540, 720}, {1, 600, 800}}, {{1, 540, 720}, {1, 540, 720}}, make([]WeeklyHours, 29)} {
		if _, err := normalizeSchedule(LocationScheduleInput{WeeklyHours: hours, ClosedDates: []string{}}); err != ErrInvalidCatalog {
			t.Fatal("invalid hours accepted", hours, err)
		}
	}
	for _, dates := range [][]string{{"2027-02-29"}, {"0000-01-01"}, {"2026-2-01"}, {"2026-01-01T00:00:00Z"}, {"2026-12-25", "2026-12-25"}, make([]string, 129)} {
		if _, err := normalizeSchedule(LocationScheduleInput{WeeklyHours: []WeeklyHours{}, ClosedDates: dates}); err != ErrInvalidCatalog {
			t.Fatal("invalid dates accepted", dates, err)
		}
	}
	empty := LocationScheduleInput{WeeklyHours: []WeeklyHours{}, ClosedDates: []string{}}
	if out, err := normalizeSchedule(empty); err != nil || !reflect.DeepEqual(out, empty) {
		t.Fatal("explicit closure rejected", out, err)
	}
	if _, err := normalizeSchedule(LocationScheduleInput{}); err != ErrInvalidCatalog {
		t.Fatal("missing fields accepted")
	}
}
func TestSchedulingRuleBounds(t *testing.T) {
	tx := &businessTestTransactions{}
	s := NewBookingCatalogService(NewBusinessService(tx))
	id := pgtype.UUID{Valid: true}
	for _, in := range []SchedulingRulesInput{{-1, 0, 0}, {1441, 0, 0}, {0, -1, 0}, {0, 1441, 0}, {0, 0, -1}, {0, 0, 525601}} {
		if _, err := s.PutSchedulingRules(context.Background(), 1, id, id, in); err != ErrInvalidCatalog {
			t.Fatal(in, err)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid input reached database")
	}
}
