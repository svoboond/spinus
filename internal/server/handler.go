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

func (s *Server) HandleForbidden(w http.ResponseWriter, r *http.Request) {
	const tmplData = "403 Forbidden"
	w.WriteHeader(http.StatusForbidden)
	s.renderTemplate(w, r, errorTmplName, tmplData)
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
	formData := &SignUpFormData{}
	formData.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		formData.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, formData)
		return
	}

	ctx := r.Context()

	username := r.PostFormValue("username")
	formData.Username = username
	parsedUsername, err := parseUsername(username)
	if err == nil {
		_, err := s.queries.GetUserByUsername(ctx, parsedUsername)
		if err == nil {
			formData.Errors["Username"] = "Username is already taken."
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		formData.Errors["Username"] = err.Error()
	}

	email := r.PostFormValue("email")
	formData.Email = email
	parsedEmail, err := parseEmail(email)
	if err == nil {
		_, err := s.queries.GetUserByEmail(ctx, parsedEmail)
		if err == nil {
			formData.Errors["Email"] = "Email is already assigned to another account."
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		formData.Errors["Email"] = err.Error()
	}

	parsedPassword, passwordErr := parsePassword(r.PostFormValue("password"))
	if passwordErr != nil {
		formData.Errors["Password"] = passwordErr.Error()
	}
	parsedRepeatPassword, repeatPasswordErr := parsePassword(
		r.PostFormValue("repeat-password"))
	if repeatPasswordErr != nil {
		formData.Errors["RepeatPassword"] = repeatPasswordErr.Error()
	}
	if passwordErr == nil &&
		repeatPasswordErr == nil &&
		parsedPassword != parsedRepeatPassword {

		formData.Errors["Password"] = "Passwords do not match."
	}

	if len(formData.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, formData)
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
	formData := &LogInFormData{}
	formData.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		formData.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, formData)
		return
	}

	username := r.PostFormValue("username")
	formData.Username = username
	parsedUsername, err := parseUsername(username)
	if err != nil {
		formData.Errors["Username"] = err.Error()
	}

	password := r.PostFormValue("password")
	formData.Password = password
	parsedPassword, err := parsePassword(password)
	if err != nil {
		formData.Errors["Password"] = err.Error()
	}

	if len(formData.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, formData)
		return
	}

	ctx := r.Context()
	user, err := s.queries.GetUser(
		ctx, spinusdb.GetUserParams{Username: parsedUsername, Crypt: parsedPassword})
	if err != nil {
		if err == pgx.ErrNoRows {
			formData.Errors["General"] = "Wrong username or password."
			s.renderTemplate(w, r, tmplName, formData)
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

	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	// TODO - list main meters where user is associated with sub meter
	mainMeters, err := s.queries.ListMainMeters(r.Context(), userId)
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
	formData := MainMeterFormData{}
	formData.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		formData.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, formData)
		return
	}

	meterId := r.PostFormValue("meter-identification")
	formData.MeterId = meterId
	parsedMeterId, err := parseMainMeterId(meterId)
	if err != nil {
		formData.Errors["MeterId"] = err.Error()
	}

	energy := r.PostFormValue("energy")
	formData.Energy = energy
	parsedEnergy, err := parseEnergy(energy)
	if err != nil {
		formData.Errors["Energy"] = err.Error()
	}

	address := r.PostFormValue("address")
	formData.Address = address
	parsedAddress, err := parseAddress(address)
	if err != nil {
		formData.Errors["Address"] = err.Error()
	}

	if len(formData.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, formData)
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
		w, r, fmt.Sprintf("/main-meter/%d/general", mainMeter.ID), http.StatusSeeOther)
}

func GetMainMeterId(r *http.Request) (int32, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "mainMeterId"), 10, 32)
	return int32(id), err
}

func (s *Server) HandleGetMainMeterGeneral(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterGeneral"

	mainMeterId, err := GetMainMeterId(r)
	if err != nil {
		s.HandleNotFound(w, r)
		return
	}
	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.GetMainMeter(r.Context(), mainMeterId)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.HandleNotFound(w, r)
			return
		}
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	// TODO - sub-meter users should also be allowed
	if userId != mainMeter.FkUser {
		s.HandleForbidden(w, r)
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		MainMeterGeneralTmplData{
			MainMeter: mainMeter,
			Upper:     MainMeterTmplData{ID: mainMeter.ID},
		},
	)
}

func (s *Server) HandleGetSubMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterCreate"

	mainMeterId, err := GetMainMeterId(r)
	if err != nil {
		s.HandleNotFound(w, r)
		return
	}
	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.GetMainMeter(r.Context(), mainMeterId)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.HandleNotFound(w, r)
			return
		}
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	// TODO - sub-meter users should also be allowed
	if userId != mainMeter.FkUser {
		s.HandleForbidden(w, r)
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SubMeterCreateTmplData{
			SubMeterFormData: SubMeterFormData{},
			Upper:            MainMeterTmplData{ID: mainMeter.ID},
		},
	)
}

func (s *Server) HandlePostSubMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterCreate"

	mainMeterId, err := GetMainMeterId(r)
	if err != nil {
		s.HandleNotFound(w, r)
		return
	}
	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.GetMainMeter(r.Context(), mainMeterId)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.HandleNotFound(w, r)
			return
		}
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	// TODO - sub-meter users should also be allowed
	if userId != mainMeter.FkUser {
		s.HandleForbidden(w, r)
		return
	}

	tmplData := SubMeterCreateTmplData{
		SubMeterFormData: SubMeterFormData{},
		Upper:            MainMeterTmplData{ID: mainMeter.ID},
	}
	tmplData.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("error parsing form", "err", err)
		tmplData.Errors["General"] = "Bad request"
		s.templates.Render(w, tmplName, tmplData)
		return
	}

	meterId := r.PostFormValue("meter-identification")
	tmplData.MeterId = meterId
	parsedMeterId, err := parseSubMeterId(meterId)
	if err != nil {
		tmplData.Errors["MeterId"] = err.Error()
	}

	if len(tmplData.Errors) > 0 {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSubMeter(
		ctx,
		spinusdb.CreateSubMeterParams{
			FkMainMeter: mainMeter.ID,
			FkUser:      userId,
			MeterID:     parsedMeterId,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r,
		fmt.Sprintf("/main-meter/%d/sub-meter/list", mainMeter.ID), http.StatusSeeOther,
	)
}

func (s *Server) HandleGetSubMeterList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterList"

	mainMeterId, err := GetMainMeterId(r)
	if err != nil {
		s.HandleNotFound(w, r)
		return
	}
	ctx := r.Context()
	userId, ok := GetUserId(ctx)
	if ok == false {
		slog.Error("error getting user ID", "userId", userId)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.GetMainMeter(r.Context(), mainMeterId)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.HandleNotFound(w, r)
			return
		}
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	// TODO - sub-meter users should also be allowed
	if userId != mainMeter.FkUser {
		s.HandleForbidden(w, r)
		return
	}

	subMeters, err := s.queries.ListSubMeters(r.Context(), mainMeterId)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	s.renderTemplate(
		w, r,
		tmplName,
		SubMeterListTmplData{
			SubMeters: subMeters,
			Upper:     MainMeterTmplData{ID: mainMeterId},
		},
	)
}
