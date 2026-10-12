package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
	"time"
)

type replayTestStore struct {
	catalogTestStore
	booking db.Booking
}

func (s replayTestStore) GetBookingReplay(context.Context, db.GetBookingReplayParams) (db.Booking, error) {
	return s.booking, nil
}
func TestBookingValidationAndCommitFailure(t *testing.T) {
	id := pgtype.UUID{Valid: true}
	input := BookingInput{ServiceID: id, LocationID: id, StartsAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)}
	tx := &businessTestTransactions{}
	svc := NewBookingService(NewBusinessService(tx))
	for _, key := range []string{"", "white space", "a,b", strings.Repeat("x", 129), "?"} {
		if _, e := svc.Create(context.Background(), 1, id, key, input); e != ErrInvalidBooking {
			t.Fatal(key, e)
		}
	}
	for _, change := range []func(*BookingInput){func(in *BookingInput) { in.ServiceID.Valid = false }, func(in *BookingInput) { in.LocationID.Valid = false }, func(in *BookingInput) { in.StartsAt = time.Time{} }, func(in *BookingInput) { in.StartsAt = in.StartsAt.Add(time.Second) }, func(in *BookingInput) { in.StartsAt = in.StartsAt.Add(time.Nanosecond) }} {
		invalid := input
		change(&invalid)
		if _, e := svc.Create(context.Background(), 1, id, "key", invalid); e != ErrInvalidBooking {
			t.Fatal(e)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid request reached transaction")
	}
	data, _ := json.Marshal(input)
	hash := sha256.Sum256(data)
	tx.store = replayTestStore{booking: db.Booking{ID: id, RequestHash: hex.EncodeToString(hash[:])}}
	tx.commitError = errors.New("private commit error")
	if result, e := svc.Create(context.Background(), 1, id, "key", input); e != ErrBookingUnavailable || result.ID.Valid || result.Replayed {
		t.Fatal("commit failure exposed result", result, e)
	}
}
