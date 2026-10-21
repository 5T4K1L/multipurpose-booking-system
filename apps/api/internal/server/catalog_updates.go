package server

import (
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
)

func serveCatalogUpdate(w http.ResponseWriter, r *http.Request, s *service.BookingCatalogService, actor int64, business, id pgtype.UUID, kind string) {
	ctx := r.Context()
	switch kind {
	case "locations":
		in, ok := decodeCatalogJSON[struct {
			Name     string `json:"name"`
			Timezone string `json:"timezone"`
		}](w, r)
		if !ok {
			return
		}
		v, err := s.UpdateLocation(ctx, actor, business, id, service.LocationInput{Name: in.Name, Timezone: in.Timezone})
		writeCatalogResult(w, publicLocation(v), 200, "", err)
	case "services":
		in, ok := decodeCatalogJSON[struct {
			Name     string `json:"name"`
			Duration int32  `json:"duration_minutes"`
			AllDay   bool   `json:"all_day"`
		}](w, r)
		if !ok {
			return
		}
		v, err := s.UpdateService(ctx, actor, business, id, service.ServiceInput{Name: in.Name, DurationMinutes: in.Duration, AllDay: in.AllDay})
		writeCatalogResult(w, publicService(v), 200, "", err)
	case "staff":
		in, ok := decodeCatalogJSON[struct {
			Name     string `json:"name"`
			RoleName string `json:"role_name"`
		}](w, r)
		if !ok {
			return
		}
		v, err := s.UpdateStaffProfile(ctx, actor, business, id, service.StaffProfileInput{Name: in.Name, RoleName: in.RoleName})
		writeCatalogResult(w, publicStaff(v), 200, "", err)
	case "resources":
		in, ok := decodeCatalogJSON[struct {
			Name     string `json:"name"`
			Capacity int32  `json:"capacity"`
		}](w, r)
		if !ok {
			return
		}
		v, err := s.UpdateResource(ctx, actor, business, id, service.ResourceInput{Name: in.Name, Capacity: in.Capacity})
		writeCatalogResult(w, publicResource(v), 200, "", err)
	}
}
