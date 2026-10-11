package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"testing"
)

type eligibilityTestStore struct {
	catalogTestStore
	targetError error
	writeError  error
	writes      *int
}

func (q eligibilityTestStore) GetService(context.Context, db.GetServiceParams) (db.Service, error) {
	return db.Service{}, nil
}
func (q eligibilityTestStore) GetLocation(context.Context, db.GetLocationParams) (db.Location, error) {
	return db.Location{}, q.targetError
}
func (q eligibilityTestStore) PutServiceLocation(context.Context, db.PutServiceLocationParams) error {
	*q.writes++
	return q.writeError
}
func (q eligibilityTestStore) DeleteServiceLocation(context.Context, db.DeleteServiceLocationParams) error {
	*q.writes++
	return q.writeError
}
func (q eligibilityTestStore) ListServiceLocation(context.Context, db.ListServiceLocationParams) ([]pgtype.UUID, error) {
	return []pgtype.UUID{{Bytes: [16]byte{1}, Valid: true}, {Bytes: [16]byte{2}, Valid: true}}, nil
}

func TestEligibilityTransactionFailures(t *testing.T) {
	failure := errors.New("private database details")
	id := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	for _, tc := range []struct {
		name                        string
		target, write, commit, want error
		writes                      int
	}{
		{name: "missing target", target: pgx.ErrNoRows, want: ErrCatalogNotFound},
		{name: "target read failed", target: failure, want: ErrCatalogUnavailable},
		{name: "write failed", write: failure, want: ErrCatalogUnavailable, writes: 1},
		{name: "commit failed", commit: failure, want: ErrCatalogUnavailable, writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			tx := &businessTestTransactions{store: eligibilityTestStore{targetError: tc.target, writeError: tc.write, writes: &writes}, commitError: tc.commit}
			err := NewBookingCatalogService(NewBusinessService(tx)).ChangeEligibility(context.Background(), 1, id, id, id, ServiceLocation, false)
			if err != tc.want || writes != tc.writes {
				t.Fatal(err, writes)
			}
		})
	}
	writes := 0
	tx := &businessTestTransactions{store: eligibilityTestStore{writes: &writes}, commitError: failure}
	s := NewBookingCatalogService(NewBusinessService(tx))
	page, err := s.ListEligibility(context.Background(), 1, id, id, ServiceLocation, PageRequest{Limit: 1})
	if err != ErrCatalogUnavailable || len(page.Items) != 0 || page.NextCursor != "" {
		t.Fatal("failed commit exposed page", page, err)
	}
}
func TestEligibilityInputAndArchivedTargetCleanup(t *testing.T) {
	tx := &businessTestTransactions{}
	s := NewBookingCatalogService(NewBusinessService(tx))
	id := pgtype.UUID{Valid: true}
	if err := s.ChangeEligibility(context.Background(), 1, id, id, id, EligibilityKind("unknown"), false); err != ErrInvalidCatalog {
		t.Fatal(err)
	}
	if err := s.ChangeEligibility(context.Background(), 1, id, pgtype.UUID{}, id, ServiceLocation, false); err != ErrInvalidCatalog {
		t.Fatal(err)
	}
	if tx.calls != 0 {
		t.Fatal("invalid eligibility reached storage")
	}
	writes := 0
	tx.store = eligibilityTestStore{writes: &writes, targetError: pgx.ErrNoRows}
	if err := s.ChangeEligibility(context.Background(), 1, id, id, id, ServiceLocation, true); err != nil || writes != 1 {
		t.Fatal("archived target cannot be unlinked", err, writes)
	}
}
