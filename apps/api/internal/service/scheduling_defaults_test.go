package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"multipurpose-booking-system/api/internal/db"
	"testing"
)

type defaultRulesStore struct {
	SchedulingStore
	row db.ServiceSchedulingRule
	err error
}

func (s defaultRulesStore) GetServiceSchedulingRules(context.Context, db.GetServiceSchedulingRulesParams) (db.ServiceSchedulingRule, error) {
	return s.row, s.err
}
func TestEffectiveSchedulingRules(t *testing.T) {
	ctx := context.Background()
	row, e := effectiveSchedulingRules(ctx, defaultRulesStore{err: pgx.ErrNoRows}, db.GetServiceSchedulingRulesParams{})
	if e != nil || row.BufferBeforeMinutes != 0 || row.BufferAfterMinutes != 0 || row.MinimumLeadMinutes != 0 {
		t.Fatal("missing customization must use defaults", row, e)
	}
	saved := db.ServiceSchedulingRule{BufferBeforeMinutes: 15, BufferAfterMinutes: 30, MinimumLeadMinutes: 60}
	row, e = effectiveSchedulingRules(ctx, defaultRulesStore{row: saved}, db.GetServiceSchedulingRulesParams{})
	if e != nil || row != saved {
		t.Fatal("saved limits changed", row, e)
	}
	failure := errors.New("database unavailable")
	_, e = effectiveSchedulingRules(ctx, defaultRulesStore{err: failure}, db.GetServiceSchedulingRulesParams{})
	if !errors.Is(e, failure) {
		t.Fatal("database failure treated as defaults", e)
	}
}
