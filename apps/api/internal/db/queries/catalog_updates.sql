-- name: UpdateBusinessProfile :one
UPDATE businesses SET name=$2, timezone=$3 WHERE id=$1 AND status='active' RETURNING *;

-- name: UpdateLocation :one
UPDATE locations SET name=$3, timezone=$4 WHERE business_id=$1 AND id=$2 AND status='active' RETURNING *;

-- name: UpdateService :one
UPDATE services SET name=$3, duration_minutes=$4, all_day=$5 WHERE business_id=$1 AND id=$2 AND status='active' RETURNING *;

-- name: UpdateStaffProfile :one
UPDATE staff_profiles SET name=$3,role_name=$4 WHERE business_id=$1 AND id=$2 AND status='active' RETURNING *;

-- name: UpdateResource :one
UPDATE resources SET name=$3, capacity=$4 WHERE business_id=$1 AND id=$2 AND status='active' RETURNING *;

-- name: ResourceHasBookings :one
SELECT EXISTS(SELECT 1 FROM bookings WHERE business_id=$1 AND resource_id=$2);

-- name: ArchiveLocation :execrows
UPDATE locations SET status='archived' WHERE business_id=$1 AND id=$2 AND status='active';

-- name: ArchiveService :execrows
UPDATE services SET status='archived' WHERE business_id=$1 AND id=$2 AND status='active';

-- name: ArchiveStaffProfile :execrows
WITH unlinked AS (DELETE FROM service_staff ss WHERE ss.business_id=$1 AND ss.staff_id=$2)
UPDATE staff_profiles st SET status='archived' WHERE st.business_id=$1 AND st.id=$2 AND st.status='active';

-- name: ArchiveResource :execrows
WITH unlinked AS (DELETE FROM service_resources sr WHERE sr.business_id=$1 AND sr.resource_id=$2)
UPDATE resources r SET status='archived' WHERE r.business_id=$1 AND r.id=$2 AND r.status='active';

-- name: CatalogHasUpcomingBookings :one
SELECT EXISTS(SELECT 1 FROM bookings WHERE business_id=sqlc.arg(business_id) AND status='confirmed' AND busy_ends_at>now() AND
 ((sqlc.arg(kind)::text='locations' AND location_id=sqlc.arg(item_id)::uuid) OR
 (sqlc.arg(kind)::text='services' AND service_id=sqlc.arg(item_id)::uuid) OR
 (sqlc.arg(kind)::text='staff' AND staff_id=sqlc.arg(item_id)::uuid) OR
 (sqlc.arg(kind)::text='resources' AND resource_id=sqlc.arg(item_id)::uuid)));
