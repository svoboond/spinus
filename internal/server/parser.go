package server

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

type Username string

func parseUsername(s string) (Username, error) {
	var v Username = Username(strings.TrimSpace(s))
	vLen := len(s)
	switch {
	case s == "":
		return v, errors.New("Enter username.")
	case vLen < 3:
		return v, errors.New("Enter username with at least 3 characters.")
	case vLen > 128:
		return v, errors.New("Enter username with maximum of 128 characters.")
	default:
		return v, nil
	}
}

type Email string

func parseEmail(s string) (Email, error) {
	v := Email(s)
	if v == "" {
		return v, errors.New("Enter email.")
	} else {
		if len(v) > 128 {
			return v, errors.New("Enter email with maximum of 128 characters.")
		}
		emailAddress, err := mail.ParseAddress(s)
		if err == nil && emailAddress.Address == s {
			return v, nil
		} else {
			return v, errors.New("Enter valid email.")
		}
	}
}

type Password string

func parsePassword(s string) (Password, error) {
	v := Password(s)
	vLen := len(v)
	switch {
	case v == "":
		return v, errors.New("Enter password.")
	case vLen < 8:
		return v, errors.New("Enter password with at least 8 characters.")
	case vLen > 128:
		return v, errors.New("Enter password with maximum of 128 characters.")
	default:
		return v, nil
	}
}

type MainMeterID string

func parseMainMeterID(s string) (MainMeterID, error) {
	v := MainMeterID(strings.TrimSpace(s))
	vLen := len(s)
	switch {
	case s == "":
		return v, errors.New("Enter meter identification.")
	case vLen < 3:
		return v, errors.New("Enter meter identification with at least 3 characters.")
	case vLen > 64:
		return v, errors.New("Enter meter identification with maximum of 64 characters.")
	default:
		return v, nil
	}
}

type Address string

func parseAddress(s string) (Address, error) {
	v := Address(strings.TrimSpace(s))
	vLen := len(v)
	switch {
	case v == "":
		return v, errors.New("Enter address.")
	case vLen < 8:
		return v, errors.New("Enter address with at least 8 characters.")
	case vLen > 255:
		return v, errors.New("Enter address with maximum of 255 characters.")
	default:
		return v, nil
	}
}

func parseEnergy(s string) (spinusdb.Energy, error) {
	v := spinusdb.Energy(s)
	if !v.Valid() {
		return v, errors.New("Enter valid energy.")
	}
	return v, nil
}

type CurrencyCode string

func parseCurrencyCode(s string) (CurrencyCode, error) {
	v := CurrencyCode(strings.TrimSpace(s))
	if len(v) != 3 {
		return v, errors.New("Enter currency code with 3 characters.")
	}
	return v, nil
}

type SubMeterID string

func parseSubMeterID(s string) (SubMeterID, error) {
	v := SubMeterID(strings.TrimSpace(s))
	vLen := len(s)
	switch {
	case v == "":
		return v, nil
	case vLen > 64:
		return v, errors.New("Enter meter identification with maximum of 64 characters.")
	default:
		return v, nil
	}
}

type FinancialBalance float64

func parseFinancialBalance(s string) (FinancialBalance, error) {
	var v FinancialBalance
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v, errors.New("Enter valid financial balance.")
	}
	return FinancialBalance(p), nil
}

type ReadingValue float64

func parseReadingValue(s string) (ReadingValue, error) {
	var v ReadingValue
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v, errors.New("Enter valid reading value.")
	}
	if p < 0 {
		return v, errors.New("Enter reading value that is no less than 0.")
	}
	return ReadingValue(p), nil
}

type Time struct {
	time.Time
}

func parseDate(s string) (Time, error) {
	var v Time
	p, err := time.Parse("2006-01-02", s)
	if err != nil {
		return v, errors.New("Enter valid date.")
	}
	return Time{p}, nil
}

type MaxDayDiff int32

func parseMaxDayDiff(s string) (MaxDayDiff, error) {
	var v MaxDayDiff
	if s == "" {
		return v, errors.New("Enter maximum day difference.")
	} else {
		p, err := strconv.Atoi(s)
		if err != nil {
			return v, errors.New("Enter valid maximum day difference.")
		}
		if p < 1 || p > 255 {
			return v, errors.New("Enter maximum day difference between 1 and 255.")
		}
		return MaxDayDiff(p), nil
	}
}

type ConsumedEnergyPrice float64

func parseConsumedEnergyPrice(s string) (ConsumedEnergyPrice, error) {
	var v ConsumedEnergyPrice
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v, errors.New("Enter valid consumed energy price.")
	}
	if p < 0.0 {
		return v, errors.New("Enter consumed energy price that is no less than 0.")
	}
	return ConsumedEnergyPrice(p), nil
}

type ServicePrice struct {
	Float64 float64
	Valid   bool
}

func parseServicePrice(s string) (ServicePrice, error) {
	var v ServicePrice
	if s == "" {
		return v, nil
	}
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v, errors.New("Enter valid service price.")
	}
	if p < 0.0 {
		return v, errors.New("Enter service price that is no less than 0.")
	}
	return ServicePrice{Float64: p, Valid: true}, nil
}
