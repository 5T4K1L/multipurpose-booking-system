package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrPublicationTooLarge = errors.New("publication exceeds service review limit")

type PublicDiscoveryStore interface {
	ListPublishedBusinesses(context.Context, db.ListPublishedBusinessesParams) ([]db.ListPublishedBusinessesRow, error)
	ListPublicationServices(context.Context, pgtype.UUID) ([]db.ListPublicationServicesRow, error)
	ActivePublishedServiceIDs(context.Context, db.ActivePublishedServiceIDsParams) ([]pgtype.UUID, error)
}
type PublicBusinessSummary struct {
	ID       pgtype.UUID `json:"id"`
	Name     string      `json:"name"`
	Timezone string      `json:"timezone"`
	Tagline  string      `json:"tagline"`
}

func (s *BusinessService) Discover(ctx context.Context, search string, in PageRequest) (Page[PublicBusinessSummary], error) {
	search = strings.TrimSpace(search)
	if !utf8.ValidString(search) || utf8.RuneCountInString(search) > 100 || strings.IndexFunc(search, unicode.IsControl) >= 0 {
		return Page[PublicBusinessSummary]{}, ErrInvalidPage
	}
	sum := sha256.Sum256([]byte(search))
	limit, c, err := parseCatalogPage(in, "public-businesses:"+hex.EncodeToString(sum[:]), 0, pgtype.UUID{}, pgtype.UUID{})
	if err != nil {
		return Page[PublicBusinessSummary]{}, err
	}
	var out Page[PublicBusinessSummary]
	err = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		rows, e := q.ListPublishedBusinesses(ctx, db.ListPublishedBusinessesParams{Search: search, HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(limit + 1)})
		if e != nil {
			return e
		}
		items := make([]PublicBusinessSummary, 0, len(rows))
		for _, row := range rows {
			items = append(items, PublicBusinessSummary{row.ID, row.Name, row.Timezone, row.Tagline})
		}
		out = finishCatalogPage(items, limit, c, func(v PublicBusinessSummary) pgtype.UUID { return v.ID })
		return nil
	})
	if err != nil {
		return Page[PublicBusinessSummary]{}, err
	}
	return out, nil
}
