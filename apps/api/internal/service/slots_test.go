package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustInstant(t *testing.T, value string) time.Time {
	t.Helper()
	v, e := time.Parse(time.RFC3339, value)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestSlotsDSTTransitions(t *testing.T) {
	for _, tc := range []struct {
		name, zone, date string
		want             int
	}{{"spring gap", "America/New_York", "2026-03-08", 11}, {"fall repeat", "America/New_York", "2026-11-01", 19}, {"half-hour repeat", "Australia/Lord_Howe", "2026-04-05", 17}, {"half-hour gap", "Australia/Lord_Howe", "2026-10-04", 13}, {"ordinary day", "Asia/Shanghai", "2026-03-08", 15}, {"skipped civil date", "Pacific/Apia", "2011-12-30", 0}} {
		t.Run(tc.name, func(t *testing.T) {
			zone, e := time.LoadLocation(tc.zone)
			if e != nil {
				t.Fatal(e)
			}
			day, e := time.Parse("2006-01-02", tc.date)
			if e != nil {
				t.Fatal(e)
			}
			slots, e := calculateSlots(context.Background(), day, zone, []WeeklyHours{{int32(day.Weekday()), 0, 240}}, nil, 30, SchedulingRulesInput{}, nil, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
			if e != nil || len(slots) != tc.want {
				t.Fatal("slot count", len(slots), tc.want, e)
			}
			repeated := 0
			for i, s := range slots {
				local := s.StartsAt.In(zone)
				if local.Format("2006-01-02") != tc.date || s.EndsAt.Sub(s.StartsAt) != 30*time.Minute {
					t.Fatal(s)
				}
				if local.Hour() == 1 && local.Minute() == 30 {
					repeated++
				}
				if i > 0 && !slots[i-1].StartsAt.Before(s.StartsAt) {
					t.Fatal("duplicate or unsorted instant")
				}
			}
			if tc.name == "fall repeat" && repeated != 2 {
				t.Fatal("lost repeated hour", repeated)
			}
		})
	}
}
func TestSlotsRuleIntersection(t *testing.T) {
	day := mustInstant(t, "2026-10-12T00:00:00Z")
	hours := []WeeklyHours{{1, 540, 720}}
	rules := SchedulingRulesInput{BufferBeforeMinutes: 15, BufferAfterMinutes: 15, MinimumLeadMinutes: 60}
	slots, e := calculateSlots(context.Background(), day, time.UTC, hours, nil, 60, rules, nil, mustInstant(t, "2026-10-12T09:30:00Z"))
	if e != nil || len(slots) != 2 || !slots[0].StartsAt.Equal(mustInstant(t, "2026-10-12T10:30:00Z")) || !slots[1].StartsAt.Equal(mustInstant(t, "2026-10-12T10:45:00Z")) {
		t.Fatal(slots, e)
	}
	windows := [][]AvailabilityWindow{{{mustInstant(t, "2026-10-12T10:00:00Z"), mustInstant(t, "2026-10-12T11:00:00Z")}}, {{mustInstant(t, "2026-10-12T09:00:00Z"), mustInstant(t, "2026-10-12T12:00:00Z")}}}
	slots, e = calculateSlots(context.Background(), day, time.UTC, hours, nil, 30, SchedulingRulesInput{15, 15, 0}, windows, day)
	if e != nil || len(slots) != 1 || !slots[0].StartsAt.Equal(mustInstant(t, "2026-10-12T10:15:00Z")) {
		t.Fatal("buffer/window intersection", slots, e)
	}
	for _, tc := range []struct {
		closed  map[string]bool
		hours   []WeeklyHours
		windows [][]AvailabilityWindow
	}{{map[string]bool{"2026-10-12": true}, hours, nil}, {nil, nil, nil}, {nil, hours, [][]AvailabilityWindow{{}}}} {
		slots, e = calculateSlots(context.Background(), day, time.UTC, tc.hours, tc.closed, 30, SchedulingRulesInput{}, tc.windows, day)
		if e != nil || len(slots) != 0 {
			t.Fatal("unavailable day produced slots", slots, e)
		}
	}
	full := []WeeklyHours{}
	for d := int32(0); d < 7; d++ {
		full = append(full, WeeklyHours{d, 0, 1440})
	}
	slots, e = calculateSlots(context.Background(), day, time.UTC, full, map[string]bool{"2026-10-13": true}, 120, SchedulingRulesInput{}, nil, day)
	if e != nil || len(slots) != 89 || slots[len(slots)-1].EndsAt.After(day.Add(24*time.Hour)) {
		t.Fatal("cross-midnight closure", len(slots), e)
	}
	slots, e = calculateSlots(context.Background(), day, time.UTC, full, nil, 10080, SchedulingRulesInput{}, nil, day)
	if e != nil || len(slots) != 96 {
		t.Fatal("multi-day duration truncated", len(slots), e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := calculateSlots(ctx, day, time.UTC, hours, nil, 30, SchedulingRulesInput{}, nil, day); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation ignored", e)
	}
}
func TestAvailabilityWindowValidation(t *testing.T) {
	a, b, c := mustInstant(t, "2026-10-12T09:00:00Z"), mustInstant(t, "2026-10-12T10:00:00Z"), mustInstant(t, "2026-10-12T11:00:00Z")
	in := []AvailabilityWindow{{b, c}, {a, b}}
	out, e := normalizeWindows(in)
	if e != nil || !out[0].StartsAt.Equal(a) || !in[0].StartsAt.Equal(b) || !windowCovers(out, a, c) {
		t.Fatal(out, e)
	}
	for _, in := range [][]AvailabilityWindow{nil, {{a, a}}, {{b, a}}, {{a, b}, {a, c}}, {{a, a.Add(8 * 24 * time.Hour)}}, {{a.Add(time.Second), b}}, make([]AvailabilityWindow, 33)} {
		if _, e := normalizeWindows(in); e != ErrInvalidCatalog {
			t.Fatal("bad window accepted", in, e)
		}
	}
	if out, e := normalizeWindows([]AvailabilityWindow{}); e != nil || out == nil || len(out) != 0 {
		t.Fatal("clear windows rejected", e)
	}
}

func TestWholeDayStartsDuringHoursAndContinuesOvernight(t *testing.T) {
	ctx := context.Background()
	day, _ := time.Parse("2006-01-02", "2027-03-13")
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	hours := []WeeklyHours{{Weekday: 6, StartMinute: 600, EndMinute: 1200}}
	now := day.Add(-48 * time.Hour)
	slots, err := calculateSlots(ctx, day, zone, hours, map[string]bool{}, 1440, SchedulingRulesInput{}, nil, now, true)
	if err != nil || len(slots) != 40 {
		t.Fatalf("overnight availability: %d %v", len(slots), err)
	}
	for _, slot := range slots {
		local := slot.StartsAt.In(zone)
		if local.Hour() < 10 || local.Hour() >= 20 || slot.EndsAt.Sub(slot.StartsAt) != 24*time.Hour {
			t.Fatal("invalid overnight slot", slot)
		}
	}
	closed, err := calculateSlots(ctx, day, zone, hours, map[string]bool{"2027-03-14": true}, 1440, SchedulingRulesInput{}, nil, now, true)
	if err != nil || len(closed) != 0 {
		t.Fatal("explicit closed date bypassed", err)
	}
	timed, err := calculateSlots(ctx, day, zone, hours, map[string]bool{}, 1440, SchedulingRulesInput{}, nil, now)
	if err != nil || len(timed) != 0 {
		t.Fatal("normal duration escaped opening hours", err)
	}
}

func TestWholeBusinessDay(t *testing.T) {
	day := mustInstant(t, "2027-03-13T00:00:00Z")
	zone, e := time.LoadLocation("Asia/Manila")
	if e != nil {
		t.Fatal(e)
	}
	hours := []WeeklyHours{{Weekday: 6, StartMinute: 600, EndMinute: 720}, {Weekday: 6, StartMinute: 780, EndMinute: 1200}}
	now := day.Add(-48 * time.Hour)
	slots, e := wholeBusinessDay(context.Background(), day, zone, hours, map[string]bool{}, SchedulingRulesInput{}, now)
	if e != nil || len(slots) != 1 {
		t.Fatal(slots, e)
	}
	s := slots[0]
	if s.StartsAt.In(zone).Hour() != 10 || s.EndsAt.In(zone).Hour() != 20 || s.EndsAt.Sub(s.StartsAt) != 10*time.Hour {
		t.Fatal(s)
	}
	closed, e := wholeBusinessDay(context.Background(), day, zone, hours, map[string]bool{"2027-03-13": true}, SchedulingRulesInput{}, now)
	if e != nil || len(closed) != 0 {
		t.Fatal("closed day offered", e)
	}
	late, e := wholeBusinessDay(context.Background(), day, zone, hours, map[string]bool{}, SchedulingRulesInput{}, s.StartsAt.Add(time.Minute))
	if e != nil || len(late) != 0 {
		t.Fatal("part of day offered", e)
	}
}
