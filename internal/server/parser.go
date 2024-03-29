package server

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

func parseUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	usernameLen := len(username)
	switch {
	case username == "":
		return "", errors.New("Enter username.")
	case usernameLen < 3:
		return "", errors.New("Enter username with at least 3 characters.")
	case usernameLen > 128:
		return "", errors.New("Enter username with maximum of 128 characters.")
	default:
		return username, nil
	}
}

func parseEmail(email string) (string, error) {
	if email == "" {
		return "", errors.New("Enter email.")
	} else {
		if len(email) > 128 {
			return "", errors.New("Enter email with maximum of 128 characters.")
		}
		emailAddress, err := mail.ParseAddress(email)
		if err == nil && emailAddress.Address == email {
			return email, nil
		} else {
			return "", errors.New("Enter valid email.")
		}
	}
}

func parsePassword(password string) (string, error) {
	passwordLen := len(password)
	switch {
	case password == "":
		return "", errors.New("Enter password.")
	case passwordLen < 8:
		return "", errors.New("Enter password with at least 8 characters.")
	case passwordLen > 128:
		return "", errors.New("Enter password with maximum of 128 characters.")
	default:
		return password, nil
	}
}

func parseMainMeterID(meterID string) (string, error) {
	meterID = strings.TrimSpace(meterID)
	meterIDLen := len(meterID)
	switch {
	case meterID == "":
		return "", errors.New("Enter meter identification.")
	case meterIDLen < 3:
		return "", errors.New("Enter meter identification with at least 3 characters.")
	case meterIDLen > 64:
		return "", errors.New("Enter meter identification with maximum of 64 characters.")
	default:
		return meterID, nil
	}
}

func parseAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	addressLen := len(address)
	switch {
	case address == "":
		return "", errors.New("Enter address.")
	case addressLen < 8:
		return "", errors.New("Enter address with at least 8 characters.")
	case addressLen > 255:
		return "", errors.New("Enter address with maximum of 255 characters.")
	default:
		return address, nil
	}
}

func parseEnergy(energy string) (spinusdb.Energy, error) {
	if energy == "" {
		return "", errors.New("Enter energy.")
	} else {
		energy := spinusdb.Energy(energy)
		if energy.Valid() == false {
			return "", errors.New("Enter valid energy.")
		}
		return energy, nil
	}
}

func parseSubMeterID(meterID string) (pgtype.Text, error) {
	meterID = strings.TrimSpace(meterID)
	meterIDLen := len(meterID)
	parsedMeterID := pgtype.Text{String: meterID}
	switch {
	case meterID == "":
		return parsedMeterID, nil
	case meterIDLen < 3:
		return parsedMeterID, errors.New(
			"Enter meter identification with at least 3 characters.")
	case meterIDLen > 64:
		return parsedMeterID, errors.New(
			"Enter meter identification with maximum of 64 characters.")
	default:
		parsedMeterID.Valid = true
		return parsedMeterID, nil
	}
}

func parseReadingValue(readingValue string) (float64, error) {
	rv, err := strconv.ParseFloat(readingValue, 64)
	if err != nil {
		return 0.0, errors.New("Enter valid reading value.")
	}
	return float64(rv), nil
}

func parseDate(date string) (pgtype.Date, error) {
	var parsedDate pgtype.Date
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return parsedDate, errors.New("Enter valid reading date.")
	}
	parsedDate.Time = t
	parsedDate.Valid = true
	return parsedDate, nil
}
