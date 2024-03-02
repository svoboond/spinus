// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package spinusdb

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type Energy string

const (
	EnergyElectricity Energy = "electricity"
	EnergyGas         Energy = "gas"
	EnergyWater       Energy = "water"
)

func (e *Energy) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Energy(s)
	case string:
		*e = Energy(s)
	default:
		return fmt.Errorf("unsupported scan type for Energy: %T", src)
	}
	return nil
}

type NullEnergy struct {
	Energy Energy
	Valid  bool // Valid is true if Energy is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullEnergy) Scan(value interface{}) error {
	if value == nil {
		ns.Energy, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.Energy.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullEnergy) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.Energy), nil
}

func (e Energy) Valid() bool {
	switch e {
	case EnergyElectricity,
		EnergyGas,
		EnergyWater:
		return true
	}
	return false
}

type MainMeter struct {
	ID      int32
	MeterID string
	Energy  Energy
	Address string
	FkUser  int32
}

type SpinusUser struct {
	ID       int32
	Username string
	Email    string
	Password string
}

type SubMeter struct {
	FkMainMeter int32
	Subid       int32
	MeterID     pgtype.Text
	FkUser      int32
}
