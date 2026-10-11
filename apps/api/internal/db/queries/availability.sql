-- name: ListStaffAvailabilityWindows :many
SELECT starts_at,ends_at FROM staff_availability_windows WHERE business_id=$1 AND staff_id=$2 ORDER BY starts_at;
-- name: DeleteStaffAvailabilityWindows :exec
DELETE FROM staff_availability_windows WHERE business_id=$1 AND staff_id=$2;
-- name: CreateStaffAvailabilityWindow :exec
INSERT INTO staff_availability_windows(business_id,staff_id,starts_at,ends_at) VALUES($1,$2,$3,$4);
-- name: ListResourceAvailabilityWindows :many
SELECT starts_at,ends_at FROM resource_availability_windows WHERE business_id=$1 AND resource_id=$2 ORDER BY starts_at;
-- name: DeleteResourceAvailabilityWindows :exec
DELETE FROM resource_availability_windows WHERE business_id=$1 AND resource_id=$2;
-- name: CreateResourceAvailabilityWindow :exec
INSERT INTO resource_availability_windows(business_id,resource_id,starts_at,ends_at) VALUES($1,$2,$3,$4);

-- name: CheckSlotEligibility :one
SELECT
 EXISTS(SELECT 1 FROM service_locations sl WHERE sl.business_id=sqlc.arg(business_id) AND sl.service_id=sqlc.arg(service_id) AND sl.location_id=sqlc.arg(location_id)) AS service_location,
 (NOT sqlc.arg(with_staff)::boolean OR (
 EXISTS(SELECT 1 FROM service_staff ss WHERE ss.business_id=sqlc.arg(business_id) AND ss.service_id=sqlc.arg(service_id) AND ss.staff_id=sqlc.arg(staff_id)) AND
 EXISTS(SELECT 1 FROM staff_locations stl WHERE stl.business_id=sqlc.arg(business_id) AND stl.staff_id=sqlc.arg(staff_id) AND stl.location_id=sqlc.arg(location_id)))) AS staff_eligible,
 (NOT sqlc.arg(with_resource)::boolean OR
 EXISTS(SELECT 1 FROM service_resources sr WHERE sr.business_id=sqlc.arg(business_id) AND sr.service_id=sqlc.arg(service_id) AND sr.resource_id=sqlc.arg(resource_id))) AS resource_eligible;
