package ui

import "github.com/google/uuid"

type Upper struct {
	UserLoggedIn bool
}

type MmUpperTmpl struct {
	MmID uuid.UUID
}
