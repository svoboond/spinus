package ui

import "github.com/google/uuid"

type Upper struct {
	UserLoggedIn bool
}

type MmUpperTmpl struct {
	MmID uuid.UUID
}

type SmUpperTmpl struct {
	SmID uuid.UUID
}

type MmBillUpperTmpl struct {
	MmBillID uuid.UUID
}
