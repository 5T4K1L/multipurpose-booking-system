package server

import (
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
)

func catalogSchedulingRoute(kind, relation string) bool {
	return kind == "locations" && relation == "schedule" || kind == "services" && (relation == "scheduling-rules" || relation == "booking-options" || relation == "booking-readiness")
}
func serveScheduling(w http.ResponseWriter, r *http.Request, s *service.BookingCatalogService, actor int64, business, id pgtype.UUID) {
	if r.PathValue("relation") == "booking-readiness" {
		if r.Method != "GET" {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		v, e := s.BookingReadiness(r.Context(), actor, business, id)
		writeCatalogResult(w, v, 200, "", e)
		return
	}
	if r.PathValue("relation") == "booking-options" {
		if r.Method == "GET" {
			v, e := s.GetBookingOptions(r.Context(), actor, business, id)
			writeCatalogResult(w, v, 200, "", e)
			return
		}
		in, ok := decodeCatalogJSON[struct {
			Enabled *bool  `json:"enabled"`
			Minimum *int32 `json:"minimum_minutes"`
			Maximum *int32 `json:"maximum_minutes"`
		}](w, r)
		if !ok {
			return
		}
		if in.Enabled == nil || in.Minimum == nil || in.Maximum == nil {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		v, e := s.PutBookingOptions(r.Context(), actor, business, id, service.BookingOptions{Enabled: *in.Enabled, MinimumMinutes: *in.Minimum, MaximumMinutes: *in.Maximum})
		writeCatalogResult(w, v, 200, "", e)
		return
	}
	if r.PathValue("kind") == "locations" {
		if r.Method == "GET" {
			v, err := s.GetLocationSchedule(r.Context(), actor, business, id)
			writeCatalogResult(w, v, 200, "", err)
			return
		}
		in, ok := decodeCatalogJSON[struct {
			WeeklyHours *[]struct {
				Weekday     *int32 `json:"weekday"`
				StartMinute *int32 `json:"start_minute"`
				EndMinute   *int32 `json:"end_minute"`
			} `json:"weekly_hours"`
			ClosedDates *[]string `json:"closed_dates"`
		}](w, r)
		if !ok {
			return
		}
		if in.WeeklyHours == nil || in.ClosedDates == nil {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		schedule := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{}, ClosedDates: *in.ClosedDates}
		for _, h := range *in.WeeklyHours {
			if h.Weekday == nil || h.StartMinute == nil || h.EndMinute == nil {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			schedule.WeeklyHours = append(schedule.WeeklyHours, service.WeeklyHours{Weekday: *h.Weekday, StartMinute: *h.StartMinute, EndMinute: *h.EndMinute})
		}
		v, err := s.PutLocationSchedule(r.Context(), actor, business, id, schedule)
		writeCatalogResult(w, v, 200, "", err)
		return
	}
	if r.Method == "GET" {
		v, err := s.GetSchedulingRules(r.Context(), actor, business, id)
		writeCatalogResult(w, v, 200, "", err)
		return
	}
	in, ok := decodeCatalogJSON[struct {
		Before *int32 `json:"buffer_before_minutes"`
		After  *int32 `json:"buffer_after_minutes"`
		Lead   *int32 `json:"minimum_lead_minutes"`
	}](w, r)
	if !ok {
		return
	}
	if in.Before == nil || in.After == nil || in.Lead == nil {
		catalogFailure(w, 400, "VALIDATION_ERROR")
		return
	}
	v, err := s.PutSchedulingRules(r.Context(), actor, business, id, service.SchedulingRulesInput{BufferBeforeMinutes: *in.Before, BufferAfterMinutes: *in.After, MinimumLeadMinutes: *in.Lead})
	writeCatalogResult(w, v, 200, "", err)
}
