-- name: GetMainMeter :one
SELECT * FROM main_meter
WHERE id = $1 LIMIT 1;

-- name: ListMainMeters :many
SELECT * FROM main_meter
ORDER BY meter_id;

-- name: CreateMainMeter :one
INSERT INTO main_meter (
  meter_id, address, fk_user
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: UpdateMainMeter :exec
UPDATE main_meter set
  meter_id = $2,
  address = $3,
  fk_user = $4
WHERE id = $1;

-- name: DeleteMainMeter :exec
DELETE FROM main_meter
WHERE id = $1;
