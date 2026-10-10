package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"testing"
)

type businessTestTransactions struct {
	store       BusinessStore
	commitError error
	calls       int
}

func (t *businessTestTransactions) WithinBusiness(ctx context.Context, fn func(BusinessStore) error) error {
	t.calls++
	if err := fn(t.store); err != nil {
		return err
	}
	return t.commitError
}

type businessTestStore struct {
	BusinessStore
	memberError error
}

func (b businessTestStore) GetUserByID(context.Context, int64) (db.User, error) {
	return db.User{ID: 1, Status: "active"}, nil
}
func (b businessTestStore) CreateBusiness(context.Context, db.CreateBusinessParams) (db.Business, error) {
	return db.Business{ID: pgtype.UUID{Valid: true}, Name: "fixture", Timezone: "UTC"}, nil
}
func (b businessTestStore) PutBusinessMembership(context.Context, db.PutBusinessMembershipParams) error {
	return b.memberError
}
func TestBusinessCreationFailureAndValidation(t *testing.T) {
	failure := errors.New("storage failure")
	for _, tc := range []struct {
		name           string
		member, commit error
	}{{"membership", failure, nil}, {"commit", nil, failure}} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &businessTestTransactions{store: businessTestStore{memberError: tc.member}, commitError: tc.commit}
			b, err := NewBusinessService(tx).Create(context.Background(), 1, BusinessInput{"Example", "UTC"})
			if !errors.Is(err, failure) || b.ID.Valid {
				t.Fatal("failed transaction exposed business", b, err)
			}
		})
	}
	tx := &businessTestTransactions{}
	for _, in := range []BusinessInput{{"", "UTC"}, {"Example", "Local"}, {"Example", "invalid/zone"}, {"bad\nname", "UTC"}} {
		if _, err := NewBusinessService(tx).Create(context.Background(), 1, in); err != ErrInvalidBusiness {
			t.Fatal(in, err)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid input reached storage")
	}
}
