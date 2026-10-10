package service

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
)

type EligibilityKind string

const (
	ServiceLocation EligibilityKind = "service_locations"
	ServiceStaff    EligibilityKind = "service_staff"
	ServiceResource EligibilityKind = "service_resources"
	StaffLocation   EligibilityKind = "staff_locations"
)

func validEligibility(kind EligibilityKind) bool {
	return kind == ServiceLocation || kind == ServiceStaff || kind == ServiceResource || kind == StaffLocation
}
func eligibilityParent(ctx context.Context, q BookingCatalogStore, business, parent pgtype.UUID, kind EligibilityKind) error {
	if kind == StaffLocation {
		_, err := q.GetStaffProfile(ctx, db.GetStaffProfileParams{BusinessID: business, ID: parent})
		return err
	}
	_, err := q.GetService(ctx, db.GetServiceParams{BusinessID: business, ID: parent})
	return err
}
func eligibilityTarget(ctx context.Context, q BookingCatalogStore, business, target pgtype.UUID, kind EligibilityKind) error {
	switch kind {
	case ServiceLocation, StaffLocation:
		_, err := q.GetLocation(ctx, db.GetLocationParams{BusinessID: business, ID: target})
		return err
	case ServiceStaff:
		_, err := q.GetStaffProfile(ctx, db.GetStaffProfileParams{BusinessID: business, ID: target})
		return err
	case ServiceResource:
		_, err := q.GetResource(ctx, db.GetResourceParams{BusinessID: business, ID: target})
		return err
	default:
		return ErrInvalidCatalog
	}
}

// ChangeEligibility writes only explicitly supported same-tenant links. It grants
// no membership, schedule, availability, or capacity reservation.
func (s *BookingCatalogService) ChangeEligibility(ctx context.Context, actor int64, business, parent, target pgtype.UUID, kind EligibilityKind, remove bool) error {
	if !validEligibility(kind) || !parent.Valid || !target.Valid {
		return ErrInvalidCatalog
	}
	_, err := catalogOperation(ctx, s, actor, business, true, func(q BookingCatalogStore, a TenantAccess) (struct{}, error) {
		if err := eligibilityParent(ctx, q, a.BusinessID(), parent, kind); err != nil {
			return struct{}{}, err
		}
		// Removal may clean up a link whose target was archived. It still uses the
		// authorized business and parent; it cannot modify another tenant's links.
		if !remove {
			if err := eligibilityTarget(ctx, q, a.BusinessID(), target, kind); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, changeEligibility(ctx, q, a.BusinessID(), parent, target, kind, remove)
	})
	return err
}
func (s *BookingCatalogService) ListEligibility(ctx context.Context, actor int64, business, parent pgtype.UUID, kind EligibilityKind, in PageRequest) (Page[pgtype.UUID], error) {
	if !validEligibility(kind) || !parent.Valid {
		return Page[pgtype.UUID]{}, ErrInvalidCatalog
	}
	limit, c, err := parseCatalogPage(in, string(kind), actor, business, parent)
	if err != nil {
		return Page[pgtype.UUID]{}, err
	}
	return catalogOperation(ctx, s, actor, business, false, func(q BookingCatalogStore, a TenantAccess) (Page[pgtype.UUID], error) {
		if err := eligibilityParent(ctx, q, a.BusinessID(), parent, kind); err != nil {
			return Page[pgtype.UUID]{}, err
		}
		rows, err := listEligibility(ctx, q, a.BusinessID(), parent, kind, c.After, in.Cursor != "", int32(limit+1))
		if err != nil {
			return Page[pgtype.UUID]{}, err
		}
		return finishCatalogPage(rows, limit, c, func(id pgtype.UUID) pgtype.UUID { return id }), nil
	})
}

type CatalogEligibilityStore interface {
	PutServiceLocation(context.Context, db.PutServiceLocationParams) error
	DeleteServiceLocation(context.Context, db.DeleteServiceLocationParams) error
	ListServiceLocation(context.Context, db.ListServiceLocationParams) ([]pgtype.UUID, error)
	PutServiceStaff(context.Context, db.PutServiceStaffParams) error
	DeleteServiceStaff(context.Context, db.DeleteServiceStaffParams) error
	ListServiceStaff(context.Context, db.ListServiceStaffParams) ([]pgtype.UUID, error)
	PutServiceResource(context.Context, db.PutServiceResourceParams) error
	DeleteServiceResource(context.Context, db.DeleteServiceResourceParams) error
	ListServiceResource(context.Context, db.ListServiceResourceParams) ([]pgtype.UUID, error)
	PutStaffLocation(context.Context, db.PutStaffLocationParams) error
	DeleteStaffLocation(context.Context, db.DeleteStaffLocationParams) error
	ListStaffLocation(context.Context, db.ListStaffLocationParams) ([]pgtype.UUID, error)
}

func changeEligibility(ctx context.Context, q BookingCatalogStore, business, parent, target pgtype.UUID, kind EligibilityKind, remove bool) error {
	switch kind {
	case ServiceLocation:
		if remove {
			return q.DeleteServiceLocation(ctx, db.DeleteServiceLocationParams{BusinessID: business, ParentID: parent, TargetID: target})
		}
		return q.PutServiceLocation(ctx, db.PutServiceLocationParams{BusinessID: business, ParentID: parent, TargetID: target})
	case ServiceStaff:
		if remove {
			return q.DeleteServiceStaff(ctx, db.DeleteServiceStaffParams{BusinessID: business, ParentID: parent, TargetID: target})
		}
		return q.PutServiceStaff(ctx, db.PutServiceStaffParams{BusinessID: business, ParentID: parent, TargetID: target})
	case ServiceResource:
		if remove {
			return q.DeleteServiceResource(ctx, db.DeleteServiceResourceParams{BusinessID: business, ParentID: parent, TargetID: target})
		}
		return q.PutServiceResource(ctx, db.PutServiceResourceParams{BusinessID: business, ParentID: parent, TargetID: target})
	case StaffLocation:
		if remove {
			return q.DeleteStaffLocation(ctx, db.DeleteStaffLocationParams{BusinessID: business, ParentID: parent, TargetID: target})
		}
		return q.PutStaffLocation(ctx, db.PutStaffLocationParams{BusinessID: business, ParentID: parent, TargetID: target})
	default:
		return ErrInvalidCatalog
	}
}
func listEligibility(ctx context.Context, q BookingCatalogStore, business, parent pgtype.UUID, kind EligibilityKind, after pgtype.UUID, hasCursor bool, size int32) ([]pgtype.UUID, error) {
	switch kind {
	case ServiceLocation:
		return q.ListServiceLocation(ctx, db.ListServiceLocationParams{BusinessID: business, ParentID: parent, AfterID: after, HasCursor: hasCursor, PageSize: size})
	case ServiceStaff:
		return q.ListServiceStaff(ctx, db.ListServiceStaffParams{BusinessID: business, ParentID: parent, AfterID: after, HasCursor: hasCursor, PageSize: size})
	case ServiceResource:
		return q.ListServiceResource(ctx, db.ListServiceResourceParams{BusinessID: business, ParentID: parent, AfterID: after, HasCursor: hasCursor, PageSize: size})
	case StaffLocation:
		return q.ListStaffLocation(ctx, db.ListStaffLocationParams{BusinessID: business, ParentID: parent, AfterID: after, HasCursor: hasCursor, PageSize: size})
	default:
		return nil, ErrInvalidCatalog
	}
}
