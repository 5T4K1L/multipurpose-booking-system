-- name: SetBusinessPublication :one
UPDATE businesses SET is_published=$2 WHERE id=$1 RETURNING is_published;

-- name: LockCustomerBookingBusiness :one
SELECT b.id FROM businesses b JOIN bookings k ON k.business_id=b.id
WHERE k.id=$1 AND k.customer_id=$2
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=k.customer_id AND u.status='active')
FOR UPDATE OF b;

-- name: GetCustomerBooking :one
SELECT b.*,child.id AS replacement_id,
 business.name AS business_name, service.name AS service_name, location.name AS location_name
FROM bookings b
JOIN businesses business ON business.id=b.business_id
JOIN services service ON service.business_id=b.business_id AND service.id=b.service_id
JOIN locations location ON location.business_id=b.business_id AND location.id=b.location_id
LEFT JOIN bookings child ON child.business_id=b.business_id AND child.supersedes_booking_id=b.id
WHERE b.id=$1 AND b.customer_id=$2
 AND NOT EXISTS(SELECT 1 FROM guest_booking_contacts c WHERE c.booking_id=b.id AND c.hidden)
 AND EXISTS (SELECT 1 FROM users u WHERE u.id=b.customer_id AND u.status='active');

-- name: CreateCustomerBooking :one
INSERT INTO bookings (business_id,service_id,location_id,staff_id,resource_id,created_by,customer_id,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at)
VALUES ($1,$2,$3,$4,$5,$6,$6,$7,$8,$9,$10,$11,$12) RETURNING *;

-- name: ListCustomerBookings :many
SELECT b.*,child.id AS replacement_id,
 business.name AS business_name, service.name AS service_name, location.name AS location_name
FROM bookings b
JOIN businesses business ON business.id=b.business_id
JOIN services service ON service.business_id=b.business_id AND service.id=b.service_id
JOIN locations location ON location.business_id=b.business_id AND location.id=b.location_id
LEFT JOIN bookings child ON child.business_id=b.business_id AND child.supersedes_booking_id=b.id
WHERE b.customer_id=sqlc.arg(customer_id)
 AND NOT EXISTS(SELECT 1 FROM guest_booking_contacts c WHERE c.booking_id=b.id AND c.hidden)
 AND EXISTS (SELECT 1 FROM users u WHERE u.id=b.customer_id AND u.status='active')
 AND (NOT sqlc.arg(has_cursor)::boolean OR (b.created_at,b.id)<(sqlc.arg(before_created_at)::timestamptz,sqlc.arg(before_id)::uuid))
ORDER BY b.created_at DESC,b.id DESC LIMIT sqlc.arg(page_size)::integer;
