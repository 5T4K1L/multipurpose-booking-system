package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

type BusinessPublicationStore interface {
	PublicAvailabilityStore
	PublicDiscoveryStore
	SavePublicBusinessProfile(context.Context, db.SavePublicBusinessProfileParams) error
	GetPublicBusinessProfile(context.Context, pgtype.UUID) ([]byte, error)
}

var ErrPublicationChanged = errors.New("publication preview changed")
var ErrPublicProfileNotFound = errors.New("public profile not found")

type PublicLocation struct {
	ID       pgtype.UUID `json:"id"`
	Name     string      `json:"name"`
	Timezone string      `json:"timezone"`
}
type PublicService struct {
	AllDay          bool             `json:"all_day,omitempty"`
	BookingOptions  *BookingOptions  `json:"booking_options,omitempty"`
	ID              pgtype.UUID      `json:"id"`
	Name            string           `json:"name"`
	DurationMinutes int32            `json:"duration_minutes"`
	Locations       []PublicLocation `json:"locations"`
}
type PublicBusinessProfile struct {
	ImageVersion string      `json:"image_version,omitempty"`
	ID           pgtype.UUID `json:"id"`
	Name         string      `json:"name"`
	Timezone     string      `json:"timezone"`
	BusinessSettings
	Services []PublicService `json:"services"`
}
type Publication struct {
	Published   bool                  `json:"published"`
	Profile     PublicBusinessProfile `json:"profile"`
	ReviewToken string                `json:"review_token"`
}

func publicationDraft(ctx context.Context, q BusinessStore, a TenantAccess) (Publication, error) {
	settings, e := q.GetBusinessSettings(ctx, a.BusinessID())
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return Publication{}, e
	}
	rows, e := q.ListPublicationServices(ctx, a.BusinessID())
	if e != nil {
		return Publication{}, e
	}
	if len(rows) > 1000 {
		return Publication{}, ErrPublicationTooLarge
	}
	links, e := q.ListPublicationServiceLocations(ctx, a.BusinessID())
	if e != nil {
		return Publication{}, e
	}
	if len(links) > 2000 {
		return Publication{}, ErrPublicationTooLarge
	}
	locations := map[pgtype.UUID][]PublicLocation{}
	for _, link := range links {
		locations[link.ServiceID] = append(locations[link.ServiceID], PublicLocation{link.ID, link.Name, link.Timezone})
	}
	services := make([]PublicService, 0, len(rows))
	for _, row := range rows {
		item := PublicService{ID: row.ID, Name: row.Name, DurationMinutes: row.DurationMinutes, AllDay: row.AllDay, Locations: append([]PublicLocation{}, locations[row.ID]...)}
		if row.RangeEnabled {
			item.BookingOptions = &BookingOptions{Enabled: true, MinimumMinutes: row.MinimumMinutes, MaximumMinutes: row.MaximumMinutes}
		}
		services = append(services, item)
	}
	image, e := q.GetBusinessImage(ctx, a.BusinessID())
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return Publication{}, e
	}
	profile := PublicBusinessProfile{imageVersion(image.Content), a.BusinessID(), a.business.Name, a.business.Timezone, publicBusinessSettings(settings), services}
	data, e := json.Marshal(profile)
	if e != nil {
		return Publication{}, e
	}
	sum := sha256.Sum256(data)
	_, e = q.GetPublicBusinessProfile(ctx, a.BusinessID())
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return Publication{}, e
	}
	return Publication{Published: e == nil, Profile: profile, ReviewToken: hex.EncodeToString(sum[:])}, nil
}

// Review hashes detect stale previews, not authorization. Membership is rechecked
// under the same tenant lock used by profile/settings writers on every operation.
func (s *BusinessService) Publication(ctx context.Context, actor int64, business pgtype.UUID, change *bool, token string) (Publication, error) {
	roles := []Role{RoleBusinessOwner, RoleBusinessAdmin, RoleManager, RoleStaff}
	if change != nil {
		roles = []Role{RoleBusinessOwner, RoleBusinessAdmin}
	}
	var out Publication
	e := s.WithinTenant(ctx, actor, business, roles, func(q BusinessStore, a TenantAccess) error {
		var err error
		out, err = publicationDraft(ctx, q, a)
		if errors.Is(err, ErrPublicationTooLarge) && change != nil && !*change {
			out.Profile = PublicBusinessProfile{ID: a.BusinessID(), Name: a.business.Name, Timezone: a.business.Timezone, Services: []PublicService{}}
			out.Published, err = q.SetBusinessPublication(ctx, db.SetBusinessPublicationParams{ID: a.BusinessID(), IsPublished: false})
			return err
		}
		if err != nil {
			return err
		}
		if change == nil {
			return nil
		}
		if *change {
			if token == "" {
				return ErrInvalidBusiness
			}
			if token != out.ReviewToken {
				return ErrPublicationChanged
			}
			data, err := json.Marshal(out.Profile)
			if err != nil {
				return err
			}
			if err = q.PublishBusinessImage(ctx, a.BusinessID()); err != nil {
				return err
			}
			if err = q.SavePublicBusinessProfile(ctx, db.SavePublicBusinessProfileParams{BusinessID: a.BusinessID(), Profile: data}); err != nil {
				return err
			}
		}
		out.Published, err = q.SetBusinessPublication(ctx, db.SetBusinessPublicationParams{ID: a.BusinessID(), IsPublished: *change})
		return err
	})
	if e != nil {
		return Publication{}, e
	}
	return out, nil
}

// Only a reviewed snapshot is projected. Internal settings/memberships/catalog
// are never serialized by this public read.
func (s *BusinessService) PublicProfile(ctx context.Context, id pgtype.UUID) (PublicBusinessProfile, error) {
	var out PublicBusinessProfile
	e := s.transactions.WithinBusiness(ctx, func(q BusinessStore) error {
		data, err := q.GetPublicBusinessProfile(ctx, id)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(data, &out); err != nil {
			return err
		}
		ids := make([]pgtype.UUID, 0, len(out.Services))
		for _, item := range out.Services {
			ids = append(ids, item.ID)
		}
		active, err := q.ActivePublishedServiceIDs(ctx, db.ActivePublishedServiceIDsParams{BusinessID: id, Column2: ids})
		if err != nil {
			return err
		}
		allowed := map[pgtype.UUID]bool{}
		for _, item := range active {
			allowed[item] = true
		}
		links, err := q.ListPublicationServiceLocations(ctx, id)
		if err != nil {
			return err
		}
		if len(links) > 2000 {
			return ErrPublicationTooLarge
		}
		locationAllowed := map[pgtype.UUID]map[pgtype.UUID]bool{}
		for _, link := range links {
			if locationAllowed[link.ServiceID] == nil {
				locationAllowed[link.ServiceID] = map[pgtype.UUID]bool{}
			}
			locationAllowed[link.ServiceID][link.ID] = true
		}
		visible := []PublicService{}
		for _, item := range out.Services {
			if allowed[item.ID] {
				locations := []PublicLocation{}
				for _, loc := range item.Locations {
					if locationAllowed[item.ID][loc.ID] {
						locations = append(locations, loc)
					}
				}
				item.Locations = locations
				visible = append(visible, item)
			}
		}
		out.Services = visible
		return nil
	})
	if errors.Is(e, pgx.ErrNoRows) {
		return PublicBusinessProfile{}, ErrPublicProfileNotFound
	}
	if e != nil {
		return PublicBusinessProfile{}, e
	}
	return out, nil
}
