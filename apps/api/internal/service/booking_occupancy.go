package service

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"time"
)

type BookingOccupancyStore interface {
	FilterUnoccupiedSlots(context.Context, db.FilterUnoccupiedSlotsParams) ([]int32, error)
}

// Only advisory reads filter occupancy here. Creation and rescheduling continue to
// use the database insert guard, allowing a replacement to overlap its predecessor.
func filterBookingOccupancy(ctx context.Context, q BookingCatalogStore, business, serviceID pgtype.UUID, in SlotRequest, result SlotResult, exclude ...pgtype.UUID) (SlotResult, error) {
	if len(result.Slots) == 0 || (!in.StaffID.Valid && !in.ResourceID.Valid) {
		return result, nil
	}
	rules, e := effectiveSchedulingRules(ctx, q, db.GetServiceSchedulingRulesParams{BusinessID: business, ServiceID: serviceID})
	if e != nil {
		return SlotResult{}, e
	}
	capacity := int32(1)
	if in.ResourceID.Valid {
		resource, e := q.GetResource(ctx, db.GetResourceParams{BusinessID: business, ID: in.ResourceID})
		if e != nil {
			return SlotResult{}, e
		}
		capacity = resource.Capacity
	}
	params := db.FilterUnoccupiedSlotsParams{BusinessID: business, WithStaff: in.StaffID.Valid, StaffID: in.StaffID, WithResource: in.ResourceID.Valid, ResourceID: in.ResourceID, Capacity: capacity}
	if len(exclude) > 0 {
		params.ExcludeBooking = exclude[0].Valid
		params.ExcludedID = exclude[0]
	}
	for _, slot := range result.Slots {
		params.Starts = append(params.Starts, pgtype.Timestamptz{Time: slot.StartsAt.Add(-time.Duration(rules.BufferBeforeMinutes) * time.Minute), Valid: true})
		params.Ends = append(params.Ends, pgtype.Timestamptz{Time: slot.EndsAt.Add(time.Duration(rules.BufferAfterMinutes) * time.Minute), Valid: true})
	}
	positions, e := q.FilterUnoccupiedSlots(ctx, params)
	if e != nil {
		return SlotResult{}, e
	}
	available := []Slot{}
	for _, p := range positions {
		if p < 1 || int(p) > len(result.Slots) {
			return SlotResult{}, fmt.Errorf("invalid occupancy position")
		}
		available = append(available, result.Slots[p-1])
	}
	result.Slots = available
	return result, nil
}
