-- name: LockGuestToken :exec
SELECT pg_advisory_xact_lock(hashtextextended($1, 22));
-- name: GetGuestVisitor :one
SELECT u.* FROM users u JOIN guest_visitors g ON g.user_id=u.id WHERE g.token_hash=$1 AND u.status='active';
-- name: CreateGuestVisitor :one
WITH u AS (INSERT INTO users(email,username) VALUES($1,'Guest') RETURNING *)
INSERT INTO guest_visitors(user_id,token_hash) SELECT u.id,$2 FROM u RETURNING user_id;
-- name: PutGuestContact :exec
INSERT INTO guest_booking_contacts(booking_id,name,phone,email,notes) VALUES($1,$2,$3,$4,$5)
ON CONFLICT(booking_id) DO UPDATE SET name=excluded.name,phone=excluded.phone,email=excluded.email,notes=excluded.notes;
-- name: CopyGuestContact :exec
INSERT INTO guest_booking_contacts(booking_id,name,phone,email,notes) SELECT sqlc.arg(replacement)::uuid,name,phone,email,notes FROM guest_booking_contacts WHERE booking_id=sqlc.arg(original)::uuid;
-- name: HideGuestBooking :execrows
UPDATE guest_booking_contacts c SET hidden=true FROM bookings b WHERE c.booking_id=b.id AND b.id=$1 AND b.customer_id=$2 AND b.status IN ('cancelled','completed','rescheduled');
-- name: GuestContact :one
SELECT c.* FROM guest_booking_contacts c JOIN bookings b ON b.id=c.booking_id WHERE b.id=$1 AND b.customer_id=$2 AND NOT c.hidden;
-- name: OwnedBusinessExists :one
SELECT EXISTS(SELECT 1 FROM business_memberships WHERE user_id=$1 AND role='BUSINESS_OWNER');
-- name: GetBusinessImage :one
SELECT * FROM business_images WHERE business_id=$1;
-- name: PutBusinessImage :exec
INSERT INTO business_images(business_id,content,content_type) VALUES($1,$2,$3) ON CONFLICT(business_id) DO UPDATE SET content=excluded.content,content_type=excluded.content_type;
-- name: PublishBusinessImage :exec
UPDATE business_images SET published_content=content,published_content_type=content_type WHERE business_id=$1;
-- name: GetPublicBusinessImage :one
SELECT i.published_content,i.published_content_type FROM business_images i JOIN businesses b ON b.id=i.business_id JOIN business_public_profiles p ON p.business_id=b.id WHERE b.id=$1 AND b.is_published AND b.status='active' AND p.public_confirmed AND octet_length(i.published_content)>0;
