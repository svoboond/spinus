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
