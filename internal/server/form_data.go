package server

type SignUpFormData struct {
	GeneralError        string
	Username            string
	UsernameError       string
	Email               string
	EmailError          string
	Password            string
	PasswordError       string
	RepeatPasswordError string
}

type LogInFormData struct {
	GeneralError  string
	Username      string
	UsernameError string
	Password      string
	PasswordError string
}

type MainMeterFormData struct {
	GeneralError string
	MeterID      string
	MeterIDError string
	Energy       string
	EnergyError  string
	Address      string
	AddressError string
}

type SubMeterFormData struct {
	GeneralError string
	MeterID      string
	MeterIDError string
}

type MainMeterReadingFormData struct {
	GeneralError      string
	ReadingValue      string
	ReadingValueError string
	ReadingDate       string
	ReadingDateError  string
}

type SubMeterReadingFormData struct {
	GeneralError      string
	ReadingValue      string
	ReadingValueError string
	ReadingDate       string
	ReadingDateError  string
}
