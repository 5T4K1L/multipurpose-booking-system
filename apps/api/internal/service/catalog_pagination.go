package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

type catalogCursor struct {
	Version  int         `json:"v"`
	Kind     string      `json:"k"`
	Actor    int64       `json:"a"`
	Business pgtype.UUID `json:"b"`
	Parent   pgtype.UUID `json:"p"`
	After    pgtype.UUID `json:"after"`
}

// These cursors are positions, never authorization. Every page revalidates membership.
func parseCatalogPage(in PageRequest, kind string, actor int64, business, parent pgtype.UUID) (int, catalogCursor, error) {
	limit := in.Limit
	if limit == 0 {
		limit = 100
	}
	expected := catalogCursor{Version: 1, Kind: kind, Actor: actor, Business: business, Parent: parent}
	if limit < 1 || limit > 100 || len(in.Cursor) > 1024 {
		return 0, expected, ErrInvalidPage
	}
	if in.Cursor == "" {
		return limit, expected, nil
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(in.Cursor)
	if err != nil {
		return 0, expected, ErrInvalidPage
	}
	var c catalogCursor
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return 0, expected, ErrInvalidPage
	}
	var extra any
	if d.Decode(&extra) != io.EOF || c.Version != 1 || c.Kind != kind || c.Actor != actor || c.Business != business || c.Parent != parent || !c.After.Valid {
		return 0, expected, ErrInvalidPage
	}
	return limit, c, nil
}
func finishCatalogPage[T any](rows []T, limit int, c catalogCursor, id func(T) pgtype.UUID) Page[T] {
	out := Page[T]{Items: []T{}}
	if len(rows) > limit {
		rows = rows[:limit]
		c.After = id(rows[len(rows)-1])
		data, _ := json.Marshal(c)
		out.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	out.Items = append(out.Items, rows...)
	return out
}
func catalogList[T any](ctx context.Context, s *BookingCatalogService, actor int64, business pgtype.UUID, in PageRequest, kind string, fetch func(BookingCatalogStore, TenantAccess, catalogCursor, int) ([]T, error), id func(T) pgtype.UUID) (Page[T], error) {
	limit, c, err := parseCatalogPage(in, kind, actor, business, pgtype.UUID{})
	if err != nil {
		return Page[T]{}, err
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (Page[T], error) {
		rows, err := fetch(q, a, c, limit+1)
		if err != nil {
			return Page[T]{}, err
		}
		return finishCatalogPage(rows, limit, c, id), nil
	})
}
func (s *BookingCatalogService) ListLocations(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest) (Page[db.Location], error) {
	return catalogList(ctx, s, actor, business, in, "Locations", func(q BookingCatalogStore, a TenantAccess, c catalogCursor, size int) ([]db.Location, error) {
		return q.ListLocations(ctx, db.ListLocationsParams{BusinessID: a.BusinessID(), HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(size)})
	}, func(row db.Location) pgtype.UUID { return row.ID })
}
func (s *BookingCatalogService) ListServices(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest) (Page[db.Service], error) {
	return catalogList(ctx, s, actor, business, in, "Services", func(q BookingCatalogStore, a TenantAccess, c catalogCursor, size int) ([]db.Service, error) {
		return q.ListServices(ctx, db.ListServicesParams{BusinessID: a.BusinessID(), HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(size)})
	}, func(row db.Service) pgtype.UUID { return row.ID })
}
func (s *BookingCatalogService) ListStaffProfiles(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest) (Page[db.StaffProfile], error) {
	return catalogList(ctx, s, actor, business, in, "StaffProfiles", func(q BookingCatalogStore, a TenantAccess, c catalogCursor, size int) ([]db.StaffProfile, error) {
		return q.ListStaffProfiles(ctx, db.ListStaffProfilesParams{BusinessID: a.BusinessID(), HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(size)})
	}, func(row db.StaffProfile) pgtype.UUID { return row.ID })
}
func (s *BookingCatalogService) ListResources(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest) (Page[db.Resource], error) {
	return catalogList(ctx, s, actor, business, in, "Resources", func(q BookingCatalogStore, a TenantAccess, c catalogCursor, size int) ([]db.Resource, error) {
		return q.ListResources(ctx, db.ListResourcesParams{BusinessID: a.BusinessID(), HasCursor: in.Cursor != "", AfterID: c.After, PageSize: int32(size)})
	}, func(row db.Resource) pgtype.UUID { return row.ID })
}

type CatalogListStore interface {
	ListLocations(context.Context, db.ListLocationsParams) ([]db.Location, error)
	ListServices(context.Context, db.ListServicesParams) ([]db.Service, error)
	ListStaffProfiles(context.Context, db.ListStaffProfilesParams) ([]db.StaffProfile, error)
	ListResources(context.Context, db.ListResourcesParams) ([]db.Resource, error)
}
