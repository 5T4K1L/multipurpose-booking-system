package server

import (
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/url"
	"time"
)

func catalogAvailabilityRoute(kind, relation string) bool {
	return (kind == "staff" || kind == "resources") && relation == "availability-windows" || kind == "services" && relation == "slots"
}
func serveAvailability(w http.ResponseWriter, r *http.Request, s *service.BookingCatalogService, actor int64, business, id pgtype.UUID) {
	if r.PathValue("relation") == "slots" {
		values, e := url.ParseQuery(r.URL.RawQuery)
		if e != nil {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		for key, v := range values {
			if len(v) != 1 || v[0] == "" || (key != "date" && key != "location_id" && key != "staff_id" && key != "resource_id") {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
		}
		loc, ok := catalogUUID(values.Get("location_id"))
		if !ok {
			catalogFailure(w, 400, "INVALID_LOCATION_ID")
			return
		}
		in := service.SlotRequest{Date: values.Get("date"), LocationID: loc}
		for key, dest := range map[string]*pgtype.UUID{"staff_id": &in.StaffID, "resource_id": &in.ResourceID} {
			if value := values.Get(key); value != "" {
				v, ok := catalogUUID(value)
				if !ok {
					catalogFailure(w, 400, "INVALID_CATALOG_ID")
					return
				}
				*dest = v
			}
		}
		result, e := s.CalculateSlots(r.Context(), actor, business, id, in)
		writeCatalogResult(w, result, 200, "", e)
		return
	}
	kind := r.PathValue("kind")
	if r.Method == "GET" {
		v, e := s.GetAvailabilityWindows(r.Context(), actor, business, id, kind)
		writeCatalogResult(w, v, 200, "", e)
		return
	}
	in, ok := decodeCatalogJSON[struct {
		Windows *[]struct {
			StartsAt string `json:"starts_at"`
			EndsAt   string `json:"ends_at"`
		} `json:"windows"`
	}](w, r)
	if !ok {
		return
	}
	if in.Windows == nil {
		catalogFailure(w, 400, "VALIDATION_ERROR")
		return
	}
	windows := []service.AvailabilityWindow{}
	for _, v := range *in.Windows {
		start, e1 := time.Parse(time.RFC3339, v.StartsAt)
		end, e2 := time.Parse(time.RFC3339, v.EndsAt)
		if e1 != nil || e2 != nil {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		windows = append(windows, service.AvailabilityWindow{StartsAt: start, EndsAt: end})
	}
	v, e := s.PutAvailabilityWindows(r.Context(), actor, business, id, kind, windows)
	writeCatalogResult(w, v, 200, "", e)
}
