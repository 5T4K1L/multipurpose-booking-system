package service

import (
	"context"
	"encoding/base64"
	"github.com/jackc/pgx/v5/pgtype"
	"strings"
	"testing"
)

func TestTenantPolicyFailsClosed(t *testing.T) {
	tx := &businessTestTransactions{}
	svc := NewBusinessService(tx)
	for _, roles := range [][]Role{nil, {RolePlatformAdmin}, {RoleBusinessOwner, Role("UNKNOWN")}} {
		if err := svc.WithinTenant(context.Background(), 1, pgtype.UUID{Valid: true}, roles, func(BusinessStore, TenantAccess) error { t.Fatal("invalid policy callback"); return nil }); err != ErrForbidden {
			t.Fatal(err)
		}
	}
	if tx.calls != 0 {
		t.Fatal("invalid policy reached database")
	}
}
func TestPageInputValidation(t *testing.T) {
	for _, p := range []PageRequest{{Limit: -1}, {Limit: 101}, {Cursor: "%%%"}, {Cursor: strings.Repeat("A", 1025)}} {
		if _, _, err := parsePage(p, "businesses", 1, pgtype.UUID{}); err != ErrInvalidPage {
			t.Fatal(p, err)
		}
	}
	if _, _, err := parsePage(PageRequest{Cursor: base64.RawURLEncoding.EncodeToString([]byte(`{"u":10}`))}, "memberships", 1, pgtype.UUID{}); err != ErrInvalidPage {
		t.Fatal("incomplete cursor accepted", err)
	}
	c := listCursor{Version: 1, Kind: "memberships", Actor: 1, Business: pgtype.UUID{Valid: true}, UserID: 10}
	token := encodeCursor(c)
	if _, _, err := parsePage(PageRequest{Cursor: token}, "memberships", 1, c.Business); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind     string
		actor    int64
		business pgtype.UUID
	}{{"businesses", 1, pgtype.UUID{}}, {"memberships", 2, c.Business}, {"memberships", 1, pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}} {
		if _, _, err := parsePage(PageRequest{Cursor: token}, tc.kind, tc.actor, tc.business); err != ErrInvalidPage {
			t.Fatal("cross-scope cursor accepted", tc, err)
		}
	}
}
