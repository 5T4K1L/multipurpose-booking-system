package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"io"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

var ErrInvalidPage = errors.New("invalid pagination")

type PageRequest struct {
	Limit  int
	Cursor string
}
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// Cursors are bounded selectors, not signed authorization claims. Each query
// still checks actor membership. Scope binding prevents accidental cross-list reuse.
type listCursor struct {
	Version  int         `json:"v"`
	Kind     string      `json:"k"`
	Actor    int64       `json:"a"`
	Business pgtype.UUID `json:"b"`
	Created  time.Time   `json:"t"`
	ID       pgtype.UUID `json:"i"`
	UserID   int64       `json:"u"`
}

func parsePage(in PageRequest, kind string, actor int64, business pgtype.UUID) (int, listCursor, error) {
	limit := in.Limit
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 || len(in.Cursor) > 1024 {
		return 0, listCursor{}, ErrInvalidPage
	}
	c := listCursor{Version: 1, Kind: kind, Actor: actor, Business: business}
	if in.Cursor == "" {
		return limit, c, nil
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(in.Cursor)
	if err != nil {
		return 0, c, ErrInvalidPage
	}
	c = listCursor{} // Do not accept omitted fields by inheriting expected values.
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return 0, c, ErrInvalidPage
	}
	var extra any
	if d.Decode(&extra) != io.EOF || c.Version != 1 || c.Kind != kind || c.Actor != actor || c.Business != business {
		return 0, c, ErrInvalidPage
	}
	if kind == "businesses" || kind == "customer-bookings" {
		if !c.ID.Valid || c.Created.IsZero() || c.Created.Year() < 1 || c.Created.Year() > 9999 || c.UserID != 0 {
			return 0, c, ErrInvalidPage
		}
	} else if c.UserID <= 0 || c.ID.Valid || !c.Created.IsZero() {
		return 0, c, ErrInvalidPage
	}
	return limit, c, nil
}
func encodeCursor(c listCursor) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}
func (s *BusinessService) ListPage(ctx context.Context, actor int64, in PageRequest) (Page[BusinessView], error) {
	limit, c, err := parsePage(in, "businesses", actor, pgtype.UUID{})
	if err != nil {
		return Page[BusinessView]{}, err
	}
	out := Page[BusinessView]{Items: []BusinessView{}}
	err = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if err := activeBusinessUser(ctx, q, actor); err != nil {
			return err
		}
		rows, err := q.ListUserBusinesses(ctx, db.ListUserBusinessesParams{UserID: actor, HasCursor: in.Cursor != "", AfterCreatedAt: pgtype.Timestamptz{Time: c.Created, Valid: in.Cursor != ""}, AfterID: c.ID, PageSize: int32(limit + 1)})
		if err != nil {
			return err
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		for _, b := range rows {
			if !businessRole(Role(b.Role)) {
				return ErrForbidden
			}
			out.Items = append(out.Items, BusinessView{b.ID, b.Name, b.Timezone, Role(b.Role)})
		}
		if more {
			last := rows[len(rows)-1]
			c.Created = last.CreatedAt.Time
			c.ID = last.ID
			out.NextCursor = encodeCursor(c)
		}
		return nil
	})
	if err != nil {
		return Page[BusinessView]{}, err
	}
	return out, nil
}
func (s *BusinessService) MembersPage(ctx context.Context, actor int64, id pgtype.UUID, in PageRequest) (Page[MembershipView], error) {
	limit, c, err := parsePage(in, "memberships", actor, id)
	if err != nil {
		return Page[MembershipView]{}, err
	}
	out := Page[MembershipView]{Items: []MembershipView{}}
	err = s.WithinTenant(ctx, actor, id, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		rows, err := q.ListBusinessMemberships(ctx, db.ListBusinessMembershipsParams{BusinessID: a.BusinessID(), AfterUserID: c.UserID, PageSize: int32(limit + 1)})
		if err != nil {
			return err
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		for _, m := range rows {
			out.Items = append(out.Items, MembershipView{m.UserID, Role(m.Role)})
		}
		if more {
			c.UserID = rows[len(rows)-1].UserID
			out.NextCursor = encodeCursor(c)
		}
		return nil
	})
	if err != nil {
		return Page[MembershipView]{}, err
	}
	return out, nil
}
