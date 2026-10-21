-- name: SavePublicBusinessProfile :exec
INSERT INTO business_public_profiles(business_id,profile,public_confirmed) VALUES($1,$2,true)
ON CONFLICT(business_id) DO UPDATE SET profile=excluded.profile,published_at=now(),public_confirmed=true;

-- name: GetPublicBusinessProfile :one
SELECT p.profile FROM business_public_profiles p JOIN businesses b ON b.id=p.business_id
WHERE b.id=$1 AND b.status='active' AND b.is_published AND p.public_confirmed;
