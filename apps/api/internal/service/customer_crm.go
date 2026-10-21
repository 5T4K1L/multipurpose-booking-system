package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"time"
	"unicode/utf8"
)

type CustomerCRMStore interface {
	ListBusinessCustomers(context.Context, db.ListBusinessCustomersParams) ([]db.ListBusinessCustomersRow, error)
}
type BusinessCustomer struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	BookingRecords    int64     `json:"booking_records"`
	ConfirmedBookings int64     `json:"confirmed_bookings"`
	LatestBooking     time.Time `json:"latest_booking"`
}

// A customer relationship is derived only from persisted customer ownership in
// this business. Membership and booking creator identity never imply a customer.
func (s *BusinessService) Customers(ctx context.Context, actor int64, business pgtype.UUID, in PageRequest, search string) (Page[BusinessCustomer], error) {
	search = strings.TrimSpace(search)
	if !utf8.ValidString(search) || len(search) > 200 {
		return Page[BusinessCustomer]{}, ErrInvalidBusiness
	}
	kind := fmt.Sprintf("crm:%x", sha256.Sum256([]byte(search)))
	limit, c, e := parsePage(in, kind, actor, business)
	if e != nil {
		return Page[BusinessCustomer]{}, e
	}
	out := Page[BusinessCustomer]{Items: []BusinessCustomer{}}
	e = s.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		rows, e := q.ListBusinessCustomers(ctx, db.ListBusinessCustomersParams{BusinessID: a.BusinessID(), AfterUser: c.UserID, Search: search, PageSize: int32(limit + 1)})
		if e != nil {
			return e
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		for _, v := range rows {
			out.Items = append(out.Items, BusinessCustomer{v.ID, v.Username, v.Email, v.BookingRecords, v.ConfirmedBookings, v.LatestBooking.Time.UTC()})
		}
		if more {
			c.UserID = rows[len(rows)-1].ID
			out.NextCursor = encodeCursor(c)
		}
		return nil
	})
	if e != nil {
		return Page[BusinessCustomer]{}, e
	}
	return out, nil
}
