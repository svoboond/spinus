package ui

type SignUpForm struct {
	GeneralErr        string
	Username          string
	UsernameErr       string
	Email             string
	EmailErr          string
	Password          string
	PasswordErr       string
	RepeatPasswordErr string
}

type LogInForm struct {
	GeneralErr  string
	Username    string
	UsernameErr string
	PasswordErr string
}

type MmForm struct {
	GeneralErr             string
	MeterIdentification    string
	MeterIdentificationErr string
	Energy                 string
	EnergyErr              string
	Address                string
	AddressErr             string
	CurrencyCode           string
	CurrencyCodeErr        string
}
