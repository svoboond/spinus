package server

type FormData struct {
	Errors map[string]string
}

type SignUpFormData struct {
	Username string
	Email    string
	Password string
	FormData
}

type LogInFormData struct {
	Username string
	Password string
	FormData
}

type MainMeterFormData struct {
	MeterId string
	Energy  string
	Address string
	FormData
}

type SubMeterFormData struct {
	MeterId string
	FormData
}

type MainMeterReadingFormData struct {
	ReadingValue string
	ReadingDate  string
	FormData
}

type SubMeterReadingFormData struct {
	ReadingValue string
	ReadingDate  string
	FormData
}
