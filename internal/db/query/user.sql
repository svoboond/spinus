-- name: GetUser :one
SELECT * FROM spinus_user
WHERE username = $1 AND password = crypt(sqlc.arg(password_crypt), password)
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
	TRIM(sqlc.arg(username)), sqlc.arg(email), crypt(sqlc.arg(password_crypt), gen_salt('bf'))
)
RETURNING *;
