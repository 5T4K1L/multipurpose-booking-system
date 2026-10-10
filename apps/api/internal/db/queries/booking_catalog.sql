-- name: CreateLocation :one
INSERT INTO locations(business_id,name,timezone) VALUES($1,$2,$3) RETURNING *;

-- name: GetLocation :one
SELECT * FROM locations WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateService :one
INSERT INTO services(business_id,name,duration_minutes) VALUES($1,$2,$3) RETURNING *;

-- name: GetService :one
SELECT * FROM services WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateStaffProfile :one
INSERT INTO staff_profiles(business_id,name) VALUES($1,$2) RETURNING *;

-- name: GetStaffProfile :one
SELECT * FROM staff_profiles WHERE business_id=$1 AND id=$2 AND status='active';

-- name: CreateResource :one
INSERT INTO resources(business_id,location_id,name,capacity) VALUES($1,$2,$3,$4) RETURNING *;

-- name: GetResource :one
SELECT r.* FROM resources r
JOIN locations l ON l.business_id=r.business_id AND l.id=r.location_id
WHERE r.business_id=$1 AND r.id=$2 AND r.status='active' AND l.status='active';
