-- name: PutServiceLocation :exec
INSERT INTO service_locations(business_id,service_id,location_id) VALUES(sqlc.arg(business_id),sqlc.arg(parent_id),sqlc.arg(target_id)) ON CONFLICT DO NOTHING;

-- name: DeleteServiceLocation :exec
DELETE FROM service_locations WHERE business_id=sqlc.arg(business_id) AND service_id=sqlc.arg(parent_id) AND location_id=sqlc.arg(target_id);

-- name: ListServiceLocation :many
SELECT t.id FROM service_locations e
JOIN locations t ON t.business_id=e.business_id AND t.id=e.location_id
WHERE e.business_id=sqlc.arg(business_id) AND e.service_id=sqlc.arg(parent_id)
AND t.status='active' AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: PutServiceStaff :exec
INSERT INTO service_staff(business_id,service_id,staff_id) VALUES(sqlc.arg(business_id),sqlc.arg(parent_id),sqlc.arg(target_id)) ON CONFLICT DO NOTHING;

-- name: DeleteServiceStaff :exec
DELETE FROM service_staff WHERE business_id=sqlc.arg(business_id) AND service_id=sqlc.arg(parent_id) AND staff_id=sqlc.arg(target_id);

-- name: ListServiceStaff :many
SELECT t.id FROM service_staff e
JOIN staff_profiles t ON t.business_id=e.business_id AND t.id=e.staff_id
WHERE e.business_id=sqlc.arg(business_id) AND e.service_id=sqlc.arg(parent_id)
AND t.status='active' AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: PutServiceResource :exec
INSERT INTO service_resources(business_id,service_id,resource_id) VALUES(sqlc.arg(business_id),sqlc.arg(parent_id),sqlc.arg(target_id)) ON CONFLICT DO NOTHING;

-- name: DeleteServiceResource :exec
DELETE FROM service_resources WHERE business_id=sqlc.arg(business_id) AND service_id=sqlc.arg(parent_id) AND resource_id=sqlc.arg(target_id);

-- name: ListServiceResource :many
SELECT t.id FROM service_resources e
JOIN resources t ON t.business_id=e.business_id AND t.id=e.resource_id
WHERE e.business_id=sqlc.arg(business_id) AND e.service_id=sqlc.arg(parent_id)
AND t.status='active' AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
AND EXISTS(SELECT 1 FROM locations l WHERE l.business_id=t.business_id AND l.id=t.location_id AND l.status='active')
ORDER BY t.id LIMIT sqlc.arg(page_size);

-- name: PutStaffLocation :exec
INSERT INTO staff_locations(business_id,staff_id,location_id) VALUES(sqlc.arg(business_id),sqlc.arg(parent_id),sqlc.arg(target_id)) ON CONFLICT DO NOTHING;

-- name: DeleteStaffLocation :exec
DELETE FROM staff_locations WHERE business_id=sqlc.arg(business_id) AND staff_id=sqlc.arg(parent_id) AND location_id=sqlc.arg(target_id);

-- name: ListStaffLocation :many
SELECT t.id FROM staff_locations e
JOIN locations t ON t.business_id=e.business_id AND t.id=e.location_id
WHERE e.business_id=sqlc.arg(business_id) AND e.staff_id=sqlc.arg(parent_id)
AND t.status='active' AND (NOT sqlc.arg(has_cursor)::boolean OR t.id > sqlc.arg(after_id)::uuid)
ORDER BY t.id LIMIT sqlc.arg(page_size);

