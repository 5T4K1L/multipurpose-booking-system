-- name: ListCalendarBookings :many
SELECT b.*, child.id AS replacement_id, s.name AS service_name,l.name AS location_name,coalesce(st.name,'')::text AS staff_name,coalesce(r.name,'')::text AS resource_name FROM bookings b
JOIN services s ON s.business_id=b.business_id AND s.id=b.service_id
JOIN locations l ON l.business_id=b.business_id AND l.id=b.location_id
LEFT JOIN staff_profiles st ON st.business_id=b.business_id AND st.id=b.staff_id
LEFT JOIN resources r ON r.business_id=b.business_id AND r.id=b.resource_id
LEFT JOIN bookings child ON child.business_id=b.business_id AND child.supersedes_booking_id=b.id
WHERE b.business_id=sqlc.arg(business_id) AND NOT EXISTS(SELECT 1 FROM business_booking_dismissals d WHERE d.business_id=b.business_id AND d.booking_id=b.id)
 AND (b.starts_at AT TIME ZONE sqlc.arg(timezone)::text)::date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
 AND (NOT sqlc.arg(has_cursor)::boolean OR (b.starts_at,b.id) > (sqlc.arg(after_start)::timestamptz,sqlc.arg(after_id)::uuid))
ORDER BY b.starts_at,b.id LIMIT sqlc.arg(page_size)::integer;

-- name: CalendarBookingCounts :one
SELECT count(*) AS total,
 count(*) FILTER (WHERE status='confirmed') AS confirmed,
 count(*) FILTER (WHERE status='completed') AS completed,
 count(*) FILTER (WHERE status='cancelled') AS cancelled,
 count(*) FILTER (WHERE status='rescheduled') AS rescheduled
FROM bookings b WHERE b.business_id=sqlc.arg(business_id) AND NOT EXISTS(SELECT 1 FROM business_booking_dismissals d WHERE d.business_id=b.business_id AND d.booking_id=b.id)
 AND (starts_at AT TIME ZONE sqlc.arg(timezone)::text)::date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date;
