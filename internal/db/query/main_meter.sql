-- name: GetMainMeter :one
SELECT * FROM main_meter
WHERE id = $1 LIMIT 1;

-- name: ListMainMeters :many
SELECT * FROM main_meter
ORDER BY no;

-- name: CreateMainMeter :one
INSERT INTO main_meter (
  no, address
) VALUES (
  $1, $2
)
RETURNING *;

-- name: UpdateMainMeter :exec
UPDATE main_meter
  set no = $2,
  address = $3
WHERE id = $1;

-- name: DeleteMainMeter :exec
DELETE FROM main_meter
WHERE id = $1;
