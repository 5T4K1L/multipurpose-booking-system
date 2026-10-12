package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
	"time"
)

type changeReplayStore struct {
	catalogTestStore
	change db.BookingChange
}

func (q changeReplayStore) GetBookingChange(context.Context, db.GetBookingChangeParams) (db.BookingChange, error) {
	return q.change, nil
}
func TestBookingLifecycleValidationAndCommit(t *testing.T) {
	id := pgtype.UUID{Valid: true}
	ctx := context.Background()
	tx := &businessTestTransactions{}
	s := NewBookingService(NewBusinessService(tx))
	start := time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)
	for _, key := range []string{"", "bad key", strings.Repeat("x", 129), "~reschedule:internal"} {
		if _, e := s.Cancel(ctx, 1, id, id, key); e != ErrInvalidBooking {
			t.Fatal(key, e)
		}
	}
	if _, e := s.Cancel(ctx, 1, id, pgtype.UUID{}, "key"); e != ErrInvalidBooking {
		t.Fatal(e)
	}
	for _, invalid := range []time.Time{{}, start.Add(time.Second), start.Add(time.Nanosecond), time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC)} {
		if _, e := s.Reschedule(ctx, 1, id, id, "key", invalid); e != ErrInvalidBooking {
			t.Fatal(invalid, e)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid change reached storage")
	}
	if bookingChangeHash("reschedule", id, start) != bookingChangeHash("reschedule", id, start.In(time.FixedZone("offset", 8*3600)).UTC()) {
		t.Fatal("offset normalization")
	}
	for _, action := range []string{"cancel", "reschedule"} {
		t.Run(action, func(t *testing.T) {
			inputTime := time.Time{}
			if action == "reschedule" {
				inputTime = start
			}
			snapshot, _ := json.Marshal(BookingChangeResult{Booking: BookingResult{ID: id, Status: "cancelled"}})
			tx.store = changeReplayStore{change: db.BookingChange{RequestHash: bookingChangeHash(action, id, inputTime), Response: snapshot}}
			tx.commitError = errors.New("private commit failure")
			result, e := s.change(ctx, 1, id, id, "key", action, inputTime)
			if e != ErrBookingUnavailable || result.Booking.ID.Valid || result.Replayed || result.Replacement != nil {
				t.Fatal("commit failure leaked result", result, e)
			}
			tx.commitError = nil
			result, e = s.change(ctx, 1, id, id, "key", action, inputTime)
			if e != nil || !result.Replayed || result.Booking.ID != id {
				t.Fatal(result, e)
			}
			if _, e = s.change(ctx, 1, id, id, "key", "different-action", inputTime); e != ErrIdempotencyConflict {
				t.Fatal(e)
			}
		})
	}
	for _, tc := range []struct {
		code, constraint string
		want             error
	}{
		{"23514", "bookings_lifecycle", ErrBookingState},
		{"23P01", "bookings_staff_conflict", ErrBookingConflict},
		{"23514", "private_constraint", ErrBookingUnavailable},
		{"40001", "", ErrBookingUnavailable},
	} {
		if got := bookingError(&pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint, Message: "private detail"}); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
}
