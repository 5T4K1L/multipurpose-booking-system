package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type GuestStore interface {
	LockGuestToken(context.Context, string) error
	GetGuestVisitor(context.Context, string) (db.User, error)
	CreateGuestVisitor(context.Context, db.CreateGuestVisitorParams) (int64, error)
	PutGuestContact(context.Context, db.PutGuestContactParams) error
	CopyGuestContact(context.Context, db.CopyGuestContactParams) error
	HideGuestBooking(context.Context, db.HideGuestBookingParams) (int64, error)
	GuestContact(context.Context, db.GuestContactParams) (db.GuestBookingContact, error)
	OwnedBusinessExists(context.Context, int64) (bool, error)
	GetBusinessImage(context.Context, pgtype.UUID) (db.BusinessImage, error)
	PutBusinessImage(context.Context, db.PutBusinessImageParams) error
	PublishBusinessImage(context.Context, pgtype.UUID) error
	GetPublicBusinessImage(context.Context, pgtype.UUID) (db.GetPublicBusinessImageRow, error)
}
type GuestContact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Notes string `json:"notes"`
}

func cleanGuestContact(in GuestContact) (GuestContact, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.TrimSpace(in.Email)
	in.Notes = strings.TrimSpace(in.Notes)
	if !utf8.ValidString(in.Name) || len(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 80 || strings.IndexFunc(in.Name, unicode.IsControl) >= 0 || len(in.Phone) < 5 || len(in.Phone) > 40 || strings.IndexFunc(in.Phone, func(r rune) bool { return !strings.ContainsRune("0123456789+()- .", r) }) >= 0 || len(in.Email) > 254 || !utf8.ValidString(in.Notes) || utf8.RuneCountInString(in.Notes) > 1000 {
		return in, ErrInvalidBooking
	}
	if in.Email != "" {
		v, e := mail.ParseAddress(in.Email)
		if e != nil || v.Address != in.Email {
			return in, ErrInvalidBooking
		}
	}
	return in, nil
}

// The random access capability grants customer-booking access only. No login
// credential or owner session is issued for the internal guest principal.
func (s *BusinessService) GuestVisitor(ctx context.Context, token string, create bool) (db.User, error) {
	hash, e := sessionHash(token)
	if e != nil {
		return db.User{}, ErrInvalidSession
	}
	var user db.User
	e = s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if create {
			if e := q.LockGuestToken(ctx, hash); e != nil {
				return e
			}
		}
		var e error
		user, e = q.GetGuestVisitor(ctx, hash)
		if errors.Is(e, pgx.ErrNoRows) && create {
			_, e = q.CreateGuestVisitor(ctx, db.CreateGuestVisitorParams{Email: hash + "@guest.invalid", TokenHash: hash})
			if e == nil {
				user, e = q.GetGuestVisitor(ctx, hash)
			}
		}
		return e
	})
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrInvalidSession
	}
	return user, e
}
func (s *BusinessService) GuestContact(ctx context.Context, actor int64, id pgtype.UUID, update *GuestContact) (GuestContact, error) {
	var out GuestContact
	e := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		if _, e := q.LockCustomerBookingBusiness(ctx, db.LockCustomerBookingBusinessParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}}); e != nil {
			return e
		}
		row, e := q.GuestContact(ctx, db.GuestContactParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
		if e != nil {
			return e
		}
		out = GuestContact{row.Name, row.Phone, row.Email, row.Notes}
		if update != nil {
			b, e := q.GetCustomerBooking(ctx, db.GetCustomerBookingParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
			if e != nil {
				return e
			}
			if b.Status != "confirmed" || !time.Now().Before(b.BusyStartsAt.Time) {
				return ErrBookingState
			}
			out, e = cleanGuestContact(*update)
			if e != nil {
				return e
			}
			return q.PutGuestContact(ctx, db.PutGuestContactParams{BookingID: id, Name: out.Name, Phone: out.Phone, Email: out.Email, Notes: out.Notes})
		}
		return nil
	})
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrBookingNotFound
	}
	return out, e
}
func (s *BusinessService) HideGuestBooking(ctx context.Context, actor int64, id pgtype.UUID) error {
	return s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		n, e := q.HideGuestBooking(ctx, db.HideGuestBookingParams{ID: id, CustomerID: pgtype.Int8{Int64: actor, Valid: true}})
		if e != nil {
			return e
		}
		if n == 0 {
			return ErrBookingState
		}
		return nil
	})
}
