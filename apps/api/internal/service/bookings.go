package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrInvalidBooking      = errors.New("invalid booking input")
	ErrBookingConflict     = errors.New("booking slot unavailable")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different input")
	ErrBookingUnavailable  = errors.New("booking storage unavailable")
)

type BookingStore interface {
	GetBookingReplay(context.Context, db.GetBookingReplayParams) (db.Booking, error)
	CreateBooking(context.Context, db.CreateBookingParams) (db.Booking, error)
}
type BookingService struct{ tenants *BusinessService }

func NewBookingService(tenants *BusinessService) *BookingService { return &BookingService{tenants} }

// A tenant-admin reservation; created_by is an audit actor, not a customer identity.
// One selected resource consumes one unit. Omitted staff/resource reserves neither.
type BookingInput struct {
	ServiceID  pgtype.UUID `json:"service_id"`
	LocationID pgtype.UUID `json:"location_id"`
	StaffID    pgtype.UUID `json:"staff_id"`
	ResourceID pgtype.UUID `json:"resource_id"`
	StartsAt   time.Time   `json:"starts_at"`
}
type BookingResult struct {
	ID pgtype.UUID `json:"id"`
	BookingInput
	EndsAt   time.Time `json:"ends_at"`
	Status   string    `json:"status"`
	Replayed bool      `json:"replayed"`
}

func bookingResult(b db.Booking, replay bool) BookingResult {
	return BookingResult{ID: b.ID, BookingInput: BookingInput{b.ServiceID, b.LocationID, b.StaffID, b.ResourceID, b.StartsAt.Time.UTC()}, EndsAt: b.EndsAt.Time.UTC(), Status: b.Status, Replayed: replay}
}
func validBookingKey(key string) bool {
	if len(key) < 1 || len(key) > 128 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func (s *BookingService) Create(ctx context.Context, actor int64, business pgtype.UUID, key string, in BookingInput) (BookingResult, error) {
	if !validBookingKey(key) || !in.ServiceID.Valid || !in.LocationID.Valid || in.StartsAt.Year() < 2000 || in.StartsAt.Year() > 2100 || in.StartsAt.Second() != 0 || in.StartsAt.Nanosecond() != 0 {
		return BookingResult{}, ErrInvalidBooking
	}
	in.StartsAt = in.StartsAt.UTC()
	// Stable struct serialization and UTC normalization make offset-equivalent retries identical.
	payload, _ := json.Marshal(in)
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	var out BookingResult
	err := s.tenants.WithinTenant(ctx, actor, business, []Role{RoleBusinessOwner, RoleBusinessAdmin}, func(q BusinessStore, a TenantAccess) error {
		existing, e := q.GetBookingReplay(ctx, db.GetBookingReplayParams{BusinessID: a.BusinessID(), CreatedBy: a.ActorID(), IdempotencyKey: key})
		if e == nil {
			if existing.RequestHash != hash {
				return ErrIdempotencyConflict
			}
			out = bookingResult(existing, true)
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		loc, e := q.GetLocation(ctx, db.GetLocationParams{BusinessID: a.BusinessID(), ID: in.LocationID})
		if e != nil {
			return e
		}
		zone, e := time.LoadLocation(loc.Timezone)
		if e != nil {
			return e
		}
		date := in.StartsAt.In(zone).Format("2006-01-02")
		day, e := time.Parse("2006-01-02", date)
		if e != nil {
			return e
		}
		slots, e := calculateCatalogSlots(ctx, q, a, in.ServiceID, SlotRequest{date, in.LocationID, in.StaffID, in.ResourceID}, day)
		if e != nil {
			return e
		}
		var end time.Time
		for _, slot := range slots.Slots {
			if slot.StartsAt.Equal(in.StartsAt) {
				end = slot.EndsAt
				break
			}
		}
		if end.IsZero() {
			return ErrBookingConflict
		}
		rules, e := q.GetServiceSchedulingRules(ctx, db.GetServiceSchedulingRulesParams{BusinessID: a.BusinessID(), ServiceID: in.ServiceID})
		if e != nil {
			return e
		}
		stamp := func(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
		b, e := q.CreateBooking(ctx, db.CreateBookingParams{BusinessID: a.BusinessID(), ServiceID: in.ServiceID, LocationID: in.LocationID, StaffID: in.StaffID, ResourceID: in.ResourceID, CreatedBy: a.ActorID(), IdempotencyKey: key, RequestHash: hash, StartsAt: stamp(in.StartsAt), EndsAt: stamp(end), BusyStartsAt: stamp(in.StartsAt.Add(-time.Duration(rules.BufferBeforeMinutes) * time.Minute)), BusyEndsAt: stamp(end.Add(time.Duration(rules.BufferAfterMinutes) * time.Minute))})
		if e != nil {
			return e
		}
		out = bookingResult(b, false)
		return nil
	})
	if err != nil {
		// Never publish a booking on rollback or commit failure.
		switch {
		case errors.Is(err, ErrForbidden):
			return BookingResult{}, ErrForbidden
		case errors.Is(err, ErrIdempotencyConflict):
			return BookingResult{}, ErrIdempotencyConflict
		case errors.Is(err, ErrBookingConflict), errors.Is(err, pgx.ErrNoRows):
			return BookingResult{}, ErrBookingConflict
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" && (pgErr.ConstraintName == "bookings_staff_conflict" || pgErr.ConstraintName == "bookings_resource_capacity") {
			return BookingResult{}, ErrBookingConflict
		}
		return BookingResult{}, ErrBookingUnavailable
	}
	return out, nil
}
