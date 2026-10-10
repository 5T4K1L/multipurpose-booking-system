package service

import (
	"encoding/base64"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"strings"
	"testing"
)

func TestCatalogCursorScope(t *testing.T) {
	id := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	parent := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	c := catalogCursor{Version: 1, Kind: string(ServiceLocation), Actor: 7, Business: id, Parent: parent, After: id}
	data, _ := json.Marshal(c)
	token := base64.RawURLEncoding.EncodeToString(data)
	if _, _, err := parseCatalogPage(PageRequest{Cursor: token}, c.Kind, 7, id, parent); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind             string
		actor            int64
		business, parent pgtype.UUID
	}{{"Locations", 7, id, parent}, {c.Kind, 8, id, parent}, {c.Kind, 7, parent, parent}, {c.Kind, 7, id, id}} {
		if _, _, err := parseCatalogPage(PageRequest{Cursor: token}, tc.kind, tc.actor, tc.business, tc.parent); err != ErrInvalidPage {
			t.Fatal("cross-scope cursor accepted", err)
		}
	}
	for _, in := range []PageRequest{{Limit: -1}, {Limit: 101}, {Cursor: "bad"}, {Cursor: strings.Repeat("x", 1025)}, {Cursor: base64.RawURLEncoding.EncodeToString([]byte(`{}`))}} {
		if _, _, err := parseCatalogPage(in, c.Kind, 7, id, parent); err != ErrInvalidPage {
			t.Fatal("bad page accepted", in, err)
		}
	}
}
