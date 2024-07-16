-- name: ListSmRdgs :many
SELECT	*
FROM	sm_rdg
WHERE	fk_sm = $1
ORDER BY rdg_date DESC;

-- name: CreateSmRdg :one
INSERT INTO sm_rdg (
	fk_sm, subid, rdg_val, rdg_date
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3
	FROM sm_rdg
	WHERE fk_sm = $1
RETURNING *;

-- name: GetSmRdgForDate :one
SELECT	1
FROM	sm_rdg
WHERE	fk_sm = $1 AND rdg_date = $2
LIMIT	1;
