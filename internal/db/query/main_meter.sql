-- name: GetMainMeter :one
SELECT * FROM main_meter
WHERE id = $1 LIMIT 1;

-- name: ListMainMeters :many
SELECT * FROM main_meter
WHERE fk_user = $1
ORDER BY id;

-- name: CreateMainMeter :one
INSERT INTO main_meter (
	meter_id, energy, address, fk_user
) VALUES (
	$1, $2, $3, $4
)
RETURNING *;

-- name: UpdateMainMeter :exec
UPDATE main_meter set
	meter_id = $2,
	energy = $3,
	address = $4,
	fk_user = $5
WHERE id = $1;

-- name: DeleteMainMeter :exec
DELETE FROM main_meter
WHERE id = $1;
