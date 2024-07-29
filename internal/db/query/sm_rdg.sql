-- name: ListSmRdgs :many
SELECT *
FROM sm_rdg
WHERE fk_sm = $1
ORDER BY rdg_date DESC;

-- name: CreateSmRdg :one
INSERT INTO sm_rdg (
	id, fk_sm, rdg_val, rdg_date
) VALUES (
	$1, $2, $3, $4
)
RETURNING *;

-- name: GetSmRdgForDate :one
SELECT 1
FROM sm_rdg
WHERE fk_sm = $1 AND rdg_date = $2
LIMIT 1;
