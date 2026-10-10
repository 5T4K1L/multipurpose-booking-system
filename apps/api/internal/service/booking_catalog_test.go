package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
)

type catalogTestStore struct {
	BusinessStore
	writeError error
}

func (q catalogTestStore) LockBusiness(context.Context, pgtype.UUID) (db.Business, error) {
	return db.Business{}, nil
}
func (q catalogTestStore) LockBusinessAccess(_ context.Context, p db.LockBusinessAccessParams) (db.LockBusinessAccessRow, error) {
	return db.LockBusinessAccessRow{ID: p.ID, Role: string(RoleBusinessOwner)}, nil
}
func (q catalogTestStore) CreateLocation(_ context.Context, p db.CreateLocationParams) (db.Location, error) {
	return db.Location{ID: pgtype.UUID{Valid: true}, BusinessID: p.BusinessID, Name: p.Name, Timezone: p.Timezone}, q.writeError
}
func TestCatalogValidation(t *testing.T) {
	tx := &businessTestTransactions{}
	s := NewBookingCatalogService(NewBusinessService(tx))
	ctx := context.Background()
	id := pgtype.UUID{Valid: true}
	for _, name := range []string{"", strings.Repeat("x", 101), "bad\nname", string([]byte{0xff})} {
		if _, err := s.CreateStaffProfile(ctx, 1, id, StaffProfileInput{Name: name}); err != ErrInvalidCatalog {
			t.Fatal(name, err)
		}
	}
	for _, zone := range []string{"", "Local", "not/a/zone", strings.Repeat("x", 101)} {
		if _, err := s.CreateLocation(ctx, 1, id, LocationInput{Name: "Place", Timezone: zone}); err != ErrInvalidCatalog {
			t.Fatal(zone, err)
		}
	}
	for _, minutes := range []int32{-1, 0, 10081} {
		if _, err := s.CreateService(ctx, 1, id, ServiceInput{Name: "Service", DurationMinutes: minutes}); err != ErrInvalidCatalog {
			t.Fatal(minutes, err)
		}
	}
	for _, in := range []ResourceInput{{Name: "Resource", Capacity: 1}, {Name: "Resource", LocationID: id, Capacity: 0}, {Name: "Resource", LocationID: id, Capacity: -1}} {
		if _, err := s.CreateResource(ctx, 1, id, in); err != ErrInvalidCatalog {
			t.Fatal(in, err)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid catalog input reached storage")
	}
}
func TestCatalogDoesNotExposeFailedTransaction(t *testing.T) {
	failure := errors.New("private storage details")
	for _, tc := range []struct {
		name          string
		write, commit error
	}{{"write", failure, nil}, {"commit", nil, failure}} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &businessTestTransactions{store: catalogTestStore{writeError: tc.write}, commitError: tc.commit}
			result, err := NewBookingCatalogService(NewBusinessService(tx)).CreateLocation(context.Background(), 1, pgtype.UUID{Valid: true}, LocationInput{Name: " Place ", Timezone: "UTC"})
			if err != ErrCatalogUnavailable || result.ID.Valid || result.Name != "" {
				t.Fatal("failed transaction exposed data", result, err)
			}
		})
	}
	tx := &businessTestTransactions{store: catalogTestStore{}}
	result, err := NewBookingCatalogService(NewBusinessService(tx)).CreateLocation(context.Background(), 1, pgtype.UUID{Valid: true}, LocationInput{Name: " Place ", Timezone: "Asia/Shanghai"})
	if err != nil || result.Name != "Place" {
		t.Fatal(result, err)
	}
}
