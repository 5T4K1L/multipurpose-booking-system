-- name: CreateBusiness :one
INSERT INTO businesses(name,timezone) VALUES($1,$2) RETURNING *;

-- name: ListUserBusinesses :many
SELECT b.*, m.role FROM businesses b
JOIN business_memberships m ON m.business_id=b.id
JOIN users u ON u.id=m.user_id
WHERE m.user_id=sqlc.arg(user_id) AND u.status='active' AND b.status='active'
AND (NOT sqlc.arg(has_cursor)::boolean OR (b.created_at,b.id) > (sqlc.arg(after_created_at)::timestamptz,sqlc.arg(after_id)::uuid))
ORDER BY b.created_at,b.id LIMIT sqlc.arg(page_size);

-- name: LockBusiness :one
SELECT * FROM businesses WHERE id=$1 AND status='active' FOR UPDATE;

-- name: LockBusinessAccess :one
SELECT b.*,m.role FROM businesses b
JOIN business_memberships m ON m.business_id=b.id
JOIN users u ON u.id=m.user_id
WHERE b.id=$1 AND m.user_id=$2 AND b.status='active' AND u.status='active'
FOR UPDATE OF m;

-- name: GetBusinessMembership :one
SELECT * FROM business_memberships WHERE business_id=$1 AND user_id=$2;

-- name: ListBusinessMemberships :many
SELECT * FROM business_memberships
WHERE business_id=sqlc.arg(business_id) AND user_id > sqlc.arg(after_user_id)::bigint
ORDER BY user_id LIMIT sqlc.arg(page_size);

-- name: PutBusinessMembership :exec
INSERT INTO business_memberships(business_id,user_id,role) VALUES($1,$2,$3)
ON CONFLICT(business_id,user_id) DO UPDATE SET role=EXCLUDED.role;

-- name: DeleteBusinessMembership :exec
DELETE FROM business_memberships WHERE business_id=$1 AND user_id=$2;
