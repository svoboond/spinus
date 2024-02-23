// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: user.sql

package spinusdb

import (
	"context"
)

const createUser = `-- name: CreateUser :one
INSERT INTO spinus_user (
  username, email, password
) VALUES (
  $1, $2, crypt($3, gen_salt('bf'))
)
RETURNING id, username, email, password
`

type CreateUserParams struct {
	Username string
	Email    string
	Crypt    string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (SpinusUser, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Username, arg.Email, arg.Crypt)
	var i SpinusUser
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
	)
	return i, err
}

const getUser = `-- name: GetUser :one
SELECT id, username, email, password FROM spinus_user
WHERE username = $1 AND password = crypt($2, password)
LIMIT 1
`

type GetUserParams struct {
	Username string
	Crypt    string
}

func (q *Queries) GetUser(ctx context.Context, arg GetUserParams) (SpinusUser, error) {
	row := q.db.QueryRow(ctx, getUser, arg.Username, arg.Crypt)
	var i SpinusUser
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
	)
	return i, err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT 1 FROM spinus_user
WHERE email = $1
LIMIT 1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (int32, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var column_1 int32
	err := row.Scan(&column_1)
	return column_1, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT 1 FROM spinus_user
WHERE username = $1
LIMIT 1
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (int32, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var column_1 int32
	err := row.Scan(&column_1)
	return column_1, err
}
