-- name: GetBookingView :one
SELECT b.*, child.id AS replacement_id FROM bookings b
LEFT JOIN bookings child ON child.business_id=b.business_id AND child.supersedes_booking_id=b.id
WHERE b.business_id=$1 AND b.id=$2;

-- name: ListTenantBookings :many
SELECT b.*, child.id AS replacement_id FROM bookings b
LEFT JOIN bookings child ON child.business_id=b.business_id AND child.supersedes_booking_id=b.id
WHERE b.business_id=sqlc.arg(business_id)
 AND (NOT sqlc.arg(has_cursor)::boolean OR b.id > sqlc.arg(after_id)::uuid)
ORDER BY b.id LIMIT sqlc.arg(page_size)::integer;

-- name: FilterUnoccupiedSlots :many
WITH candidates AS (
 SELECT value AS starts_at,(sqlc.arg(ends)::timestamptz[])[ordinality] AS ends_at,ordinality FROM unnest(sqlc.arg(starts)::timestamptz[]) WITH ORDINALITY AS slots(value,ordinality)
)
SELECT c.ordinality::integer AS position FROM candidates c
WHERE (NOT sqlc.arg(with_staff)::boolean OR NOT EXISTS (
 SELECT 1 FROM bookings b WHERE b.business_id=sqlc.arg(business_id) AND b.staff_id=sqlc.arg(staff_id)::uuid AND b.status='confirmed'
 AND b.busy_starts_at<c.ends_at AND b.busy_ends_at>c.starts_at
)) AND (NOT sqlc.arg(with_resource)::boolean OR NOT EXISTS (
 SELECT 1 FROM (
  SELECT sum(sum(delta)) OVER (ORDER BY instant) AS occupied FROM (
   SELECT greatest(b.busy_starts_at,c.starts_at) AS instant,1 AS delta FROM bookings b
   WHERE b.business_id=sqlc.arg(business_id) AND b.resource_id=sqlc.arg(resource_id)::uuid AND b.status='confirmed'
    AND b.busy_starts_at<c.ends_at AND b.busy_ends_at>c.starts_at
   UNION ALL
   SELECT least(b.busy_ends_at,c.ends_at),-1 FROM bookings b
   WHERE b.business_id=sqlc.arg(business_id) AND b.resource_id=sqlc.arg(resource_id)::uuid AND b.status='confirmed'
    AND b.busy_starts_at<c.ends_at AND b.busy_ends_at>c.starts_at
  ) events GROUP BY instant
 ) occupancy WHERE occupied >= sqlc.arg(capacity)::integer
)) ORDER BY c.ordinality;
