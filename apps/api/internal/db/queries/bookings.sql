-- name: GetBookingReplay :one
SELECT * FROM bookings WHERE business_id=$1 AND created_by=$2 AND idempotency_key=$3;

-- name: CreateBooking :one
INSERT INTO bookings (business_id,service_id,location_id,staff_id,resource_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;
