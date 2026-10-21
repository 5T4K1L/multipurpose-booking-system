-- name: ListPublicationServiceLocations :many
SELECT sl.service_id,l.id,l.name,l.timezone FROM service_locations sl
JOIN services s ON s.business_id=sl.business_id AND s.id=sl.service_id AND s.status='active'
JOIN locations l ON l.business_id=sl.business_id AND l.id=sl.location_id AND l.status='active'
WHERE sl.business_id=$1 ORDER BY sl.service_id,l.id LIMIT 2001;

-- name: PublicAvailabilityCandidates :many
WITH staff_choices AS (
 SELECT st.id FROM service_staff ss
 JOIN staff_profiles st ON st.business_id=ss.business_id AND st.id=ss.staff_id AND st.status='active'
 WHERE ss.business_id=sqlc.arg(business_id) AND ss.service_id=sqlc.arg(service_id)
 AND (NOT EXISTS(SELECT 1 FROM staff_locations sl WHERE sl.business_id=st.business_id AND sl.staff_id=st.id)
 OR EXISTS(SELECT 1 FROM staff_locations sl WHERE sl.business_id=st.business_id AND sl.staff_id=st.id AND sl.location_id=sqlc.arg(location_id)))
 UNION ALL SELECT NULL::uuid WHERE NOT EXISTS(SELECT 1 FROM service_staff WHERE business_id=sqlc.arg(business_id) AND service_id=sqlc.arg(service_id))
),resource_choices AS (
 SELECT r.id FROM service_resources sr
 JOIN resources r ON r.business_id=sr.business_id AND r.id=sr.resource_id AND r.status='active' AND r.location_id=sqlc.arg(location_id)
 WHERE sr.business_id=sqlc.arg(business_id) AND sr.service_id=sqlc.arg(service_id)
 UNION ALL SELECT NULL::uuid WHERE NOT EXISTS(SELECT 1 FROM service_resources sr JOIN resources r ON r.business_id=sr.business_id AND r.id=sr.resource_id WHERE sr.business_id=sqlc.arg(business_id) AND sr.service_id=sqlc.arg(service_id) AND r.location_id=sqlc.arg(location_id))
)
SELECT s.id AS staff_id,r.id AS resource_id FROM staff_choices s CROSS JOIN resource_choices r
ORDER BY s.id,r.id LIMIT 33;
