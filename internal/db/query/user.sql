-- name: GetUser :one
SELECT * FROM spinus_user
WHERE username = $1 AND password = crypt($2, password)
LIMIT 1;

-- name: GetUserByUsername :one
SELECT 1 FROM spinus_user
WHERE username = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT 1 FROM spinus_user
WHERE email = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO spinus_user (
	username, email, password
) VALUES (
	$1, $2, crypt($3, gen_salt('bf'))
)
RETURNING *;
