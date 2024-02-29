package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

const errorTmplName = "error"

type Upper struct {
	UserLoggedIn bool
}

func (s *Server) HandleInternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	s.renderTemplate(w, r, errorTmplName, err.Error())
}

func (s *Server) renderTemplate(
	w http.ResponseWriter, r *http.Request, name string, data any) error {

	const upperTmplName = "upper"

	var buf bytes.Buffer
	var userLoggedIn bool
	if r.Context().Value(userIdKey) != emptyUserIdValue {
		userLoggedIn = true
	}
	upperData := Upper{UserLoggedIn: userLoggedIn}
	if err := s.templates.Render(&buf, upperTmplName, upperData); err != nil {
		slog.Error("error rendering template", "template", upperTmplName, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	if err := s.templates.Render(&buf, name, data); err != nil {
		slog.Error("error rendering template", "template", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	buf.WriteTo(w)
	return nil
}

func (s *Server) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	const tmplData = "404 Page Not Found"
	w.WriteHeader(http.StatusNotFound)
	s.renderTemplate(w, r, errorTmplName, tmplData)
}

func (s *Server) HandleNotAllowed(w http.ResponseWriter, r *http.Request) {
	const tmplData = "405 Method Not Allowed"
	w.WriteHeader(http.StatusMethodNotAllowed)
	s.renderTemplate(w, r, errorTmplName, tmplData)
}

func (s *Server) HandleGetIndex(w http.ResponseWriter, r *http.Request) {
	const tmplName = "index"
	s.renderTemplate(w, r, tmplName, nil)
}

func (s *Server) HandleGetSignUp(w http.ResponseWriter, r *http.Request) {
	const tmplName = "signUp"
	s.renderTemplate(w, r, tmplName, nil)
}

func (s *Server) HandleGetLogIn(w http.ResponseWriter, r *http.Request) {
	const tmplName = "logIn"
	s.renderTemplate(w, r, tmplName, nil)
}

func (s *Server) HandlePostSignUp(w http.ResponseWriter, r *http.Request) {
	const tmplName = "signUp"
	form := &SignUpForm{}
	form.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		form.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, form)
		return
	}

	ctx := r.Context()

	username := r.PostFormValue("username")
	form.Username = username
	parsedUsername, err := parseUsername(username)
	if err == nil {
		_, err := s.queries.GetUserByUsername(ctx, parsedUsername)
		if err == nil {
			form.Errors["Username"] = "Username is already taken."
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		form.Errors["Username"] = err.Error()
	}

	email := r.PostFormValue("email")
	form.Email = email
	parsedEmail, err := parseEmail(email)
	if err == nil {
		_, err := s.queries.GetUserByEmail(ctx, parsedEmail)
		if err == nil {
			form.Errors["Email"] = "Email is already assigned to another account."
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		form.Errors["Email"] = err.Error()
	}

	parsedPassword, passwordErr := parsePassword(r.PostFormValue("password"))
	if passwordErr != nil {
		form.Errors["Password"] = passwordErr.Error()
	}
	parsedRepeatPassword, repeatPasswordErr := parsePassword(
		r.PostFormValue("repeat-password"))
	if repeatPasswordErr != nil {
		form.Errors["RepeatPassword"] = repeatPasswordErr.Error()
	}
	if passwordErr == nil &&
		repeatPasswordErr == nil &&
		parsedPassword != parsedRepeatPassword {

		form.Errors["Password"] = "Passwords do not match."
	}

	if len(form.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, form)
		return
	}

	user, err := s.queries.CreateUser(
		ctx,
		spinusdb.CreateUserParams{
			Username: parsedUsername,
			Email:    parsedEmail,
			Crypt:    parsedPassword,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	if err := s.sessionManager.RenewToken(ctx); err != nil {
		slog.Error("error renewing token", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.sessionManager.Put(ctx, "userId", user.ID)

	query := r.URL.Query()
	next := query.Get("next")
	if next != "" {
		query.Del("next")
		redirectUrl := url.URL{Path: next, RawQuery: query.Encode()}
		http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) HandlePostLogOut(w http.ResponseWriter, r *http.Request) {
	const tmplName = "logOut"

	ctx := r.Context()
	if err := s.sessionManager.Destroy(ctx); err != nil {
		slog.Error("error destroying token", "err", err)
		ctx = context.WithValue(ctx, userIdKey, emptyUserIdValue)
		s.HandleInternalServerError(w, r.WithContext(ctx), err)
		return
	}
	ctx = context.WithValue(ctx, userIdKey, emptyUserIdValue)
	s.renderTemplate(w, r.WithContext(ctx), tmplName, nil)
}

func (s *Server) HandlePostLogIn(w http.ResponseWriter, r *http.Request) {
	const tmplName = "logIn"
	form := &LogInForm{}
	form.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		form.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, form)
		return
	}

	username := r.PostFormValue("username")
	form.Username = username
	parsedUsername, err := parseUsername(username)
	if err != nil {
		form.Errors["Username"] = err.Error()
	}

	password := r.PostFormValue("password")
	form.Password = password
	parsedPassword, err := parsePassword(password)
	if err != nil {
		form.Errors["Password"] = err.Error()
	}

	if len(form.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, form)
		return
	}

	ctx := r.Context()
	user, err := s.queries.GetUser(
		ctx, spinusdb.GetUserParams{Username: parsedUsername, Crypt: parsedPassword})
	if err != nil {
		if err == pgx.ErrNoRows {
			form.Errors["General"] = "Wrong username or password."
			s.renderTemplate(w, r, tmplName, form)
		} else {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
		}
		return
	}
	if err := s.sessionManager.RenewToken(ctx); err != nil {
		slog.Error("error renewing token", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.sessionManager.Put(ctx, "userId", user.ID)

	query := r.URL.Query()
	next := query.Get("next")
	if next != "" {
		query.Del("next")
		redirectUrl := url.URL{Path: next, RawQuery: query.Encode()}
		http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) HandleGetMainMeterList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterList"

	mainMeters, err := s.queries.ListMainMeters(r.Context())
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	s.renderTemplate(w, r, tmplName, mainMeters)
}

func (s *Server) HandleGetMainMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterCreate"
	s.renderTemplate(w, r, tmplName, nil)
}

func (s *Server) HandlePostMainMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterCreate"
	form := MainMeterForm{}
	form.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		form.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, form)
		return
	}

	meterId := r.PostFormValue("meter-identification")
	form.MeterId = meterId
	parsedMeterId, err := parseMeterId(meterId)
	if err != nil {
		form.Errors["MeterId"] = err.Error()
	}

	energy := r.PostFormValue("energy")
	form.Energy = energy
	parsedEnergy, err := parseEnergy(energy)
	if err != nil {
		form.Errors["Energy"] = err.Error()
	}

	address := r.PostFormValue("address")
	form.Address = address
	parsedAddress, err := parseAddress(address)
	if err != nil {
		form.Errors["Address"] = err.Error()
	}

	if len(form.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, form)
		return
	}

	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.CreateMainMeter(
		ctx,
		spinusdb.CreateMainMeterParams{
			MeterID: parsedMeterId,
			Energy:  parsedEnergy,
			Address: parsedAddress,
			FkUser:  userId,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r, fmt.Sprintf("/main-meter-detail/%d", mainMeter.ID), http.StatusSeeOther)
}

func (s *Server) HandleGetMainMeterDetail(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterDetail"

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		s.HandleNotFound(w, r)
		return
	}
	mainMeter, err := s.queries.GetMainMeter(r.Context(), int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			s.HandleNotFound(w, r)
			return
		}
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.renderTemplate(w, r, tmplName, mainMeter)
}
