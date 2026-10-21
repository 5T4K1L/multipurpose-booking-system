-- name: CreateLocation :one
INSERT INTO locations(business_id,name,timezone) VALUES($1,$2,$3) RETURNING *;

-- name: GetLocation :one
SELECT * FROM locations WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateService :one
INSERT INTO services(business_id,name,duration_minutes,all_day) VALUES($1,$2,$3,$4) RETURNING *;

-- name: GetService :one
SELECT * FROM services WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateStaffProfile :one
INSERT INTO staff_profiles(business_id,name,role_name) VALUES($1,$2,$3) RETURNING *;

-- name: GetStaffProfile :one
SELECT * FROM staff_profiles WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateResource :one
INSERT INTO resources(business_id,location_id,name,capacity) VALUES($1,$2,$3,$4) RETURNING *;

-- name: GetResource :one
SELECT r.* FROM resources r
JOIN locations l ON l.business_id=r.business_id AND l.id=r.location_id
WHERE r.business_id=$1 AND r.id=$2 AND r.status='active' AND l.status='active';
-- name: ListLocations :many
SELECT t.* FROM locations t
WHERE t.business_id=sqlc.arg(business_id) AND t.status='active'
AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: ListServices :many
SELECT t.* FROM services t
WHERE t.business_id=sqlc.arg(business_id) AND t.status='active'
AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: ListStaffProfiles :many
SELECT t.* FROM staff_profiles t
WHERE t.business_id=sqlc.arg(business_id) AND t.status='active'
AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: ListResources :many
SELECT t.* FROM resources t
WHERE t.business_id=sqlc.arg(business_id) AND t.status='active'
AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
AND EXISTS(SELECT 1 FROM locations l WHERE l.business_id=t.business_id AND l.id=t.location_id AND l.status='active')
ORDER BY t.id LIMIT sqlc.arg(page_size);


