-- name: GetTenantBooking :one
SELECT * FROM bookings WHERE business_id=$1 AND id=$2;

-- name: CloseBooking :one
UPDATE bookings SET status=$3, closed_by=$4, closed_at=clock_timestamp()
WHERE business_id=$1 AND id=$2 AND status='confirmed' RETURNING *;

-- name: GetBookingChange :one
SELECT * FROM booking_changes WHERE business_id=$1 AND actor_id=$2 AND idempotency_key=$3;

-- name: CreateBookingChange :exec
INSERT INTO booking_changes (business_id,actor_id,idempotency_key,booking_id,action,request_hash,response)
VALUES ($1,$2,$3,$4,$5,$6,$7);

-- name: CreateReplacementBooking :one
INSERT INTO bookings (business_id,service_id,location_id,staff_id,resource_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at,supersedes_booking_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *;
