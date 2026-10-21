package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"image"
	"image/jpeg"
	"image/png"
	"multipurpose-booking-system/api/internal/db"
)

func imageVersion(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (s *BusinessService) BusinessImage(ctx context.Context, actor int64, business pgtype.UUID, content []byte, update bool) ([]byte, string, error) {
	kind := "image/jpeg"
	if update && len(content) > 0 {
		if len(content) > 2*1024*1024 {
			return nil, "", ErrInvalidBusiness
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(content))
		if e != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 6000 || cfg.Height > 6000 || cfg.Width*cfg.Height > 24000000 {
			return nil, "", ErrInvalidBusiness
		}
		decoded, _, e := image.Decode(bytes.NewReader(content))
		if e != nil {
			return nil, "", ErrInvalidBusiness
		}
		var output bytes.Buffer
		if format == "png" {
			kind = "image/png"
			e = png.Encode(&output, decoded)
		} else {
			e = jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 85})
		}
		if e != nil || output.Len() > 2*1024*1024 {
			return nil, "", ErrInvalidBusiness
		}
		content = output.Bytes()
	}
	var out []byte
	e := s.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		if update {
			if content == nil {
				content = []byte{}
			}
			return q.PutBusinessImage(ctx, db.PutBusinessImageParams{BusinessID: business, Content: content, ContentType: kind})
		}
		row, e := q.GetBusinessImage(ctx, business)
		if e != nil {
			return e
		}
		out = row.Content
		kind = row.ContentType
		return nil
	})
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrPublicProfileNotFound
	}
	return out, kind, e
}
func (s *BusinessService) PublicBusinessImage(ctx context.Context, business pgtype.UUID) ([]byte, string, error) {
	var row db.GetPublicBusinessImageRow
	e := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error { var e error; row, e = q.GetPublicBusinessImage(ctx, business); return e })
	if errors.Is(e, pgx.ErrNoRows) {
		e = ErrPublicProfileNotFound
	}
	return row.PublishedContent, row.PublishedContentType, e
}
