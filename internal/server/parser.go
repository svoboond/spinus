package server

import (
	"errors"
	"net/mail"

	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

func parseUsername(username string) (string, error) {
	switch {
	case username == "":
		return "", errors.New("Enter username.")
	case len(username) < 3:
		return "", errors.New("Enter username with at least 3 characters.")
	default:
		return username, nil
	}
}

func parseEmail(email string) (string, error) {
	if email == "" {
		return "", errors.New("Enter email.")
	} else {
		emailAddress, err := mail.ParseAddress(email)
		if err == nil && emailAddress.Address == email {
			return email, nil
		} else {
			return "", errors.New("Enter valid email.")
		}
	}
}

func parsePassword(password string) (string, error) {
	switch {
	case password == "":
		return "", errors.New("Enter password.")
	case len(password) < 8:
		return "", errors.New("Enter password with at least 8 characters.")
	default:
		return password, nil
	}
}

func parseMeterId(meterId string) (string, error) {
	switch {
	case meterId == "":
		return "", errors.New("Enter meter identification.")
	case len(meterId) < 3:
		return "", errors.New("Enter meter identification with at least 3 characters.")
	default:
		return meterId, nil
	}
}

func parseAddress(address string) (string, error) {
	switch {
	case address == "":
		return "", errors.New("Enter address.")
	case len(address) < 8:
		return "", errors.New("Enter address with at least 8 characters.")
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
