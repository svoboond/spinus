package server

type form struct {
	Errors map[string]string
}

type SignUpForm struct {
	Username string
	Email    string
	Password string
	form
}

type LogInForm struct {
	Username string
	Password string
	form
}

type MainMeterForm struct {
	MeterId string
	Energy  string
	Address string
	form
}

type SubMeterForm struct {
	MeterId string
	form
}
