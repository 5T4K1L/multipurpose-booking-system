-- name: ListPublishedBusinesses :many
SELECT b.id,coalesce(p.profile->>'name','')::text AS name,coalesce(p.profile->>'timezone','')::text AS timezone,coalesce(p.profile->>'tagline','')::text AS tagline
FROM businesses b JOIN business_public_profiles p ON p.business_id=b.id
WHERE b.status='active' AND b.is_published AND p.public_confirmed
 AND p.profile ? 'services'
 AND (sqlc.arg(search)::text='' OR strpos(lower(p.profile->>'name'),lower(sqlc.arg(search)::text))>0
  OR strpos(lower(p.profile->>'tagline'),lower(sqlc.arg(search)::text))>0)
 AND (NOT sqlc.arg(has_cursor)::boolean OR b.id>sqlc.arg(after_id)::uuid)
ORDER BY b.id LIMIT sqlc.arg(page_size)::integer;

-- name: ListPublicationServices :many
SELECT s.id,s.name,s.duration_minutes,s.all_day,(coalesce(o.enabled,false) AND NOT s.all_day)::boolean AS range_enabled,
 coalesce(o.minimum_minutes,s.duration_minutes)::integer AS minimum_minutes,
 coalesce(o.maximum_minutes,s.duration_minutes)::integer AS maximum_minutes
FROM services s LEFT JOIN service_booking_options o ON o.business_id=s.business_id AND o.service_id=s.id
WHERE s.business_id=$1 AND s.status='active' ORDER BY s.id LIMIT 1001;

-- name: ActivePublishedServiceIDs :many
SELECT id FROM services WHERE business_id=$1 AND status='active' AND id=ANY($2::uuid[]);
