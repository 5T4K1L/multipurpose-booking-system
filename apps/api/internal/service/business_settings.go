package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"net/mail"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

type BusinessSettingsStore interface {
	GetBusinessSettings(context.Context, pgtype.UUID) (db.BusinessSetting, error)
	PutBusinessSettings(context.Context, db.PutBusinessSettingsParams) (db.BusinessSetting, error)
}
type BusinessSettings struct {
	Tagline      string `json:"tagline"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Website      string `json:"website"`
}

func publicBusinessSettings(v db.BusinessSetting) BusinessSettings {
	return BusinessSettings{v.Tagline, v.Description, v.ContactEmail, v.ContactPhone, v.Website}
}
func normalizeBusinessSettings(in BusinessSettings) (BusinessSettings, error) {
	fields := []struct {
		value     *string
		max       int
		multiline bool
	}{{&in.Tagline, 120, false}, {&in.Description, 2000, true}, {&in.ContactEmail, 254, false}, {&in.ContactPhone, 40, false}, {&in.Website, 500, false}}
	for _, field := range fields {
		*field.value = strings.TrimSpace(*field.value)
		if !utf8.ValidString(*field.value) || utf8.RuneCountInString(*field.value) > field.max || strings.IndexFunc(*field.value, func(r rune) bool { return unicode.IsControl(r) && !(field.multiline && (r == '\n' || r == '\t')) }) >= 0 {
			return BusinessSettings{}, ErrInvalidBusiness
		}
	}
	if in.ContactEmail != "" {
		address, e := mail.ParseAddress(in.ContactEmail)
		if e != nil || address.Address != in.ContactEmail {
			return BusinessSettings{}, ErrInvalidBusiness
		}
	}
	if in.Website != "" {
		u, e := url.Parse(in.Website)
		if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return BusinessSettings{}, ErrInvalidBusiness
		}
	}
	return in, nil
}
func (s *BusinessService) Settings(ctx context.Context, actor int64, business pgtype.UUID) (BusinessSettings, error) {
	var out BusinessSettings
	err := s.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin, RoleManager, RoleStaff}, func(q BusinessStore, a TenantAccess) error {
		row, e := q.GetBusinessSettings(ctx, a.BusinessID())
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		out = publicBusinessSettings(row)
		return nil
	})
	if err != nil {
		return BusinessSettings{}, err
	}
	return out, nil
}
func (s *BusinessService) SaveSettings(ctx context.Context, actor int64, business pgtype.UUID, in BusinessSettings) (BusinessSettings, error) {
	in, err := normalizeBusinessSettings(in)
	if err != nil {
		return BusinessSettings{}, err
	}
	var out BusinessSettings
	err = s.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		row, e := q.PutBusinessSettings(ctx, db.PutBusinessSettingsParams{BusinessID: a.BusinessID(), Tagline: in.Tagline, Description: in.Description, ContactEmail: in.ContactEmail, ContactPhone: in.ContactPhone, Website: in.Website})
		if e != nil {
			return e
		}
		out = publicBusinessSettings(row)
		return nil
	})
	if err != nil {
		return BusinessSettings{}, err
	}
	return out, nil
}
