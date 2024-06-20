package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (s *Server) renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {

	const upperTmplName = "upper"

	var buf bytes.Buffer
	var userLoggedIn bool
	if r.Context().Value(userIDKey) != emptyUserIDValue {
		userLoggedIn = true
	}
	upperData := Upper{UserLoggedIn: userLoggedIn}
	if err := s.templates.Render(&buf, upperTmplName, upperData); err != nil {
		slog.Error("error rendering template", "template", upperTmplName, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.templates.Render(&buf, name, data); err != nil {
		slog.Error("error rendering template", "template", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err := buf.WriteTo(w)
	if err != nil {
		slog.Error("error writing to buffer", "template", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

	formData := SignUpFormData{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, formData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	ctx := r.Context()

	iUsername := r.PostFormValue("username")
	formData.Username = iUsername
	username, err := parseUsername(iUsername)
	if err == nil {
		_, err := s.queries.GetUserByUsername(ctx, string(username))
		if err == nil {
			formData.UsernameError = "Username is already taken."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		formData.UsernameError = err.Error()
		formError = true
	}

	iEmail := r.PostFormValue("email")
	formData.Email = iEmail
	email, err := parseEmail(iEmail)
	if err == nil {
		_, err := s.queries.GetUserByEmail(ctx, string(email))
		if err == nil {
			formData.EmailError = "Email is already assigned to another account."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		formData.EmailError = err.Error()
		formError = true
	}

	password, passwordErr := parsePassword(r.PostFormValue("password"))
	if passwordErr != nil {
		formData.PasswordError = passwordErr.Error()
		formError = true
	}
	repeatPassword, repeatPasswordErr := parsePassword(r.PostFormValue("repeat-password"))
	if repeatPasswordErr != nil {
		formData.RepeatPasswordError = repeatPasswordErr.Error()
		formError = true
	}
	if passwordErr == nil &&
		repeatPasswordErr == nil &&
		password != repeatPassword {

		formData.PasswordError = "Passwords do not match."
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, formData)
		return
	}

	user, err := s.queries.CreateUser(
		ctx,
		spinusdb.CreateUserParams{
			Username:      string(username),
			Email:         string(email),
			PasswordCrypt: string(password),
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
	s.sessionManager.Put(ctx, "userID", user.ID)

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
		ctx = context.WithValue(ctx, userIDKey, emptyUserIDValue)
		s.HandleInternalServerError(w, r.WithContext(ctx), err)
		return
	}
	ctx = context.WithValue(ctx, userIDKey, emptyUserIDValue)
	s.renderTemplate(w, r.WithContext(ctx), tmplName, nil)
}

func (s *Server) HandlePostLogIn(w http.ResponseWriter, r *http.Request) {
	const tmplName = "logIn"
	formData := LogInFormData{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, formData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iUsername := r.PostFormValue("username")
	formData.Username = iUsername
	username, err := parseUsername(iUsername)
	if err != nil {
		formData.UsernameError = err.Error()
		formError = true
	}

	iPassword := r.PostFormValue("password")
	formData.Password = iPassword
	password, err := parsePassword(iPassword)
	if err != nil {
		formData.PasswordError = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, formData)
		return
	}

	ctx := r.Context()
	user, err := s.queries.GetUser(
		ctx, spinusdb.GetUserParams{
			Username: string(username), PasswordCrypt: string(password),
		},
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			formData.GeneralError = "Wrong username or password."
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
	s.sessionManager.Put(ctx, "userID", user.ID)

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
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeters, err := s.queries.ListUserMainMeters(r.Context(), userID)
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
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, formData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iMeterID := r.PostFormValue("meter-identification")
	formData.MeterID = iMeterID
	meterID, err := parseMainMeterID(iMeterID)
	if err != nil {
		formData.MeterIDError = err.Error()
		formError = true
	}

	iEnergy := r.PostFormValue("energy")
	formData.Energy = iEnergy
	energy, err := parseEnergy(iEnergy)
	if err != nil {
		formData.EnergyError = err.Error()
		formError = true
	}

	iAddress := r.PostFormValue("address")
	formData.Address = iAddress
	address, err := parseAddress(iAddress)
	if err != nil {
		formData.AddressError = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, formData)
		return
	}

	ctx := r.Context()
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.CreateMainMeter(
		ctx,
		spinusdb.CreateMainMeterParams{
			MeterID: string(meterID),
			Energy:  energy,
			Address: string(address),
			FkUser:  userID,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r, fmt.Sprintf("/main-meter/%d/overview", mainMeter.ID), http.StatusSeeOther)
}

func (s *Server) HandleGetMainMeterOverview(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterOverview"

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		MainMeterOverviewTmplData{
			GetMainMeterRow: mainMeter,
			Upper:           MainMeterTmplData{ID: mainMeter.ID},
		},
	)
}

func (s *Server) HandleGetSubMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterCreate"

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
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

	ctx := r.Context()
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	tmplData := SubMeterCreateTmplData{
		SubMeterFormData: SubMeterFormData{},
		Upper:            MainMeterTmplData{ID: mainMeter.ID},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iMeterID := r.PostFormValue("meter-identification")
	tmplData.MeterID = iMeterID
	subMeterID, err := parseSubMeterID(iMeterID)
	if err != nil {
		tmplData.MeterIDError = err.Error()
		formError = true
	}

	IFinancialBalance := r.PostFormValue("financial-balance")
	tmplData.FinancialBalance = IFinancialBalance
	financialBalance, err := parseFinancialBalance(IFinancialBalance)
	if err != nil {
		tmplData.FinancialBalanceError = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSubMeter(
		ctx,
		spinusdb.CreateSubMeterParams{
			FkMainMeter:      mainMeter.ID,
			MeterID:          pgtype.Text{String: string(subMeterID), Valid: true},
			FinancialBalance: float64(financialBalance),
			FkUser:           userID,
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

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	mainMeterID := mainMeter.ID
	subMeters, err := s.queries.ListSubMeters(r.Context(), mainMeterID)
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
			Upper:     MainMeterTmplData{ID: mainMeterID},
		},
	)
}

func (s *Server) HandleGetSubMeterOverview(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterOverview"

	ctx := r.Context()
	subMeter, ok := GetSubMeter(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", subMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}

	s.renderTemplate(
		w, r,
		tmplName,
		SubMeterOverviewTmplData{
			GetSubMeterRow: subMeter,
			Upper: SubMeterTmplData{
				MainMeterID: subMeter.MainMeterID, Subid: subMeter.Subid},
		},
	)
}

func (s *Server) HandleGetSubMeterReadingList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterReadingList"

	ctx := r.Context()
	subMeter, ok := GetSubMeter(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", subMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}
	subMeterSubid := subMeter.Subid
	subMeterReadings, err := s.queries.ListSubMeterReadings(r.Context(), subMeterSubid)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SubMeterReadingListTmplData{
			SubMeterReadings: subMeterReadings,
			Upper: SubMeterTmplData{
				MainMeterID: subMeter.MainMeterID, Subid: subMeterSubid},
		},
	)
}

func (s *Server) HandleGetSubMeterReadingCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterReadingCreate"

	ctx := r.Context()
	subMeter, ok := GetSubMeter(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", subMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SubMeterReadingCreateTmplData{
			SubMeterReadingFormData: SubMeterReadingFormData{},
			Upper: SubMeterTmplData{
				MainMeterID: subMeter.MainMeterID, Subid: subMeter.Subid},
		},
	)
}

func (s *Server) HandlePostSubMeterReadingCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "subMeterReadingCreate"

	ctx := r.Context()
	subMeter, ok := GetSubMeter(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", subMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}

	mainMeterID := subMeter.MainMeterID
	subMeterSubid := subMeter.Subid
	tmplData := SubMeterReadingCreateTmplData{
		SubMeterReadingFormData: SubMeterReadingFormData{},
		Upper: SubMeterTmplData{
			MainMeterID: mainMeterID, Subid: subMeterSubid},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iReadingVal := r.PostFormValue("reading-value")
	tmplData.ReadingValue = iReadingVal
	readingVal, err := parseReadingValue(iReadingVal)
	if err != nil {
		tmplData.ReadingValueError = err.Error()
		formError = true
	}

	subMeterID := subMeter.ID
	iReadingDate := r.PostFormValue("reading-date")
	tmplData.ReadingDate = iReadingDate
	readingTime, err := parseDate(iReadingDate)
	readingDate := pgtype.Date{Time: readingTime.Time, Valid: true}
	if err == nil {
		_, err = s.queries.GetSubMeterReadingForDate(
			ctx,
			spinusdb.GetSubMeterReadingForDateParams{
				FkSubMeter:  subMeterID,
				ReadingDate: readingDate,
			},
		)
		if err == nil {
			tmplData.ReadingDateError = "Reading for the given date already exists."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		tmplData.ReadingDateError = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSubMeterReading(
		ctx,
		spinusdb.CreateSubMeterReadingParams{
			FkSubMeter:   subMeter.ID,
			ReadingValue: float64(readingVal),
			ReadingDate:  readingDate,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r,
		fmt.Sprintf(
			"/main-meter/%d/sub-meter/%d/reading/list", mainMeterID, subMeterSubid),
		http.StatusSeeOther,
	)
}

func (s *Server) HandleGetMainMeterBillingList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterBillingList"

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	mainMeterID := mainMeter.ID
	s.renderTemplate(
		w, r, tmplName,
		MainMeterBillingListTmplData{Upper: MainMeterTmplData{ID: mainMeterID}},
	)
}

func (s *Server) HandleGetMainMeterBillingCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterBillingCreate"

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		MainMeterBillingCreateTmplData{
			MainMeterBillingFormData: NewMainMeterBillingFormData(),
			Upper:                    MainMeterTmplData{ID: mainMeter.ID},
		},
	)
}

func (s *Server) HandlePostMainMeterBillingCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterBillingCreate"

	ctx := r.Context()
	mainMeter, ok := GetMainMeter(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	mainMeterID := mainMeter.ID

	var mainMeterBillingPeriodForms []*MainMeterBillingPeriodFormData
	var subMeterBillingForms []*MainMeterBillingSubMeterFormData
	tmplData := MainMeterBillingCreateTmplData{
		MainMeterBillingFormData: MainMeterBillingFormData{
			MainMeterBillingPeriods: mainMeterBillingPeriodForms,
			SubMeterBillings:        subMeterBillingForms,
		},
		Upper: MainMeterTmplData{ID: mainMeterID},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	var addBillingPeriod bool
	if r.PostFormValue("add-billing-period") != "" {
		addBillingPeriod = true
	}
	var removeBillingPeriod bool
	if r.PostFormValue("remove-billing-period") != "" {
		removeBillingPeriod = true
	}
	var parse bool
	if !addBillingPeriod && !removeBillingPeriod {
		parse = true
	}

	mainMeterBilling := spinusdb.CreateMainMeterBillingParams{FkMainMeter: mainMeterID}

	iMaxDayDiff := r.PostFormValue("max-day-diff")
	tmplData.MaxDayDiff = iMaxDayDiff
	maxDayDiff, err := parseMaxDayDiff(iMaxDayDiff)
	dayDiff := int(maxDayDiff)
	if err != nil {
		tmplData.MaxDayDiffError = err.Error()
		formError = true
	}
	mainMeterBilling.MaxDayDiff = int32(maxDayDiff)

	// From latest to earliest.
	var mainMeterBillingPeriods []*spinusdb.CreateMainMeterBillingPeriodParams
	var calcBreakPoints BreakPoints // From latest to earliest.

	iBeginDates := r.PostForm["begin-date"]
	iEndDates := r.PostForm["end-date"]
	iBeginReadingVals := r.PostForm["begin-reading-value"]
	iEndReadingVals := r.PostForm["end-reading-value"]
	iConsumedEnergyPrices := r.PostForm["consumed-energy-price"]
	iServicePrices := r.PostForm["service-price"]

	billingPeriodsLen := len(iBeginDates)
	if billingPeriodsLen == 0 {
		tmplData.GeneralError = "No billing period provided."
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}
	billingPeriodsLastIndex := billingPeriodsLen - 1
	var mainMeterBillingMinTime time.Time // Shifted one day back.
	var mainMeterBillingMaxTime time.Time

	mmBillingPeriodIndex := 0
	for i := billingPeriodsLastIndex; i >= 0; i-- {
		mainMeterBillingPeriodForm := &MainMeterBillingPeriodFormData{}
		tmplData.MainMeterBillingPeriods = append(
			tmplData.MainMeterBillingPeriods, mainMeterBillingPeriodForm)
		iBeginDate := iBeginDates[i]
		mainMeterBillingPeriodForm.BeginDate = iBeginDate
		iEndDate := iEndDates[i]
		mainMeterBillingPeriodForm.EndDate = iEndDate
		iBeginReadingVal := iBeginReadingVals[i]
		mainMeterBillingPeriodForm.BeginReadingValue = iBeginReadingVal
		iEndReadingVal := iEndReadingVals[i]
		mainMeterBillingPeriodForm.EndReadingValue = iEndReadingVal
		iConsumedEnergyPrice := iConsumedEnergyPrices[i]
		mainMeterBillingPeriodForm.ConsumedEnergyPrice = iConsumedEnergyPrice
		iServicePrice := iServicePrices[i]
		mainMeterBillingPeriodForm.ServicePrice = iServicePrice
		if parse {
			mainMeterBillingPeriod := &spinusdb.CreateMainMeterBillingPeriodParams{}
			mainMeterBillingPeriods = append(
				mainMeterBillingPeriods, mainMeterBillingPeriod)
			beginTime, err := parseDate(iBeginDate)
			if err != nil {
				mainMeterBillingPeriodForm.BeginDateError = err.Error()
				formError = true
			}
			endTime, err := parseDate(iEndDate)
			if err != nil {
				mainMeterBillingPeriodForm.EndDateError = err.Error()
				formError = true
			}
			beginReadingVal, err := parseReadingValue(iBeginReadingVal)
			if err != nil {
				mainMeterBillingPeriodForm.BeginReadingValueError = err.Error()
				formError = true
			}
			endReadingVal, err := parseReadingValue(iEndReadingVal)
			if err != nil {
				mainMeterBillingPeriodForm.EndReadingValueError = err.Error()
				formError = true
			}
			consumedEnergyPrice, err := parseConsumedEnergyPrice(iConsumedEnergyPrice)
			if err != nil {
				mainMeterBillingPeriodForm.ConsumedEnergyPriceError = err.Error()
				formError = true
			}
			servicePrice, err := parseServicePrice(iServicePrice)
			if err != nil {
				mainMeterBillingPeriodForm.ServicePriceError = err.Error()
				formError = true
			}
			if !formError {
				if i != billingPeriodsLastIndex {
					laterIndex := mmBillingPeriodIndex - 1
					previousBeginDate :=
						mainMeterBillingPeriods[laterIndex].BeginDate
					if endTime.AddDate(0, 0, 1) !=
						previousBeginDate.Time {

						laterBillingPeriod :=
							tmplData.
								MainMeterBillingPeriods[laterIndex]
						laterBillingPeriod.BeginDateError =
							"Begin date must follow previous " +
								"billing period's end date."
						formError = true
						mmBillingPeriodIndex++
						continue
					}
				}
				mainMeterBillingPeriod.BeginDate = pgtype.Date{
					Time: beginTime.Time, Valid: true}
				if endTime.Before(beginTime.Time) {
					mainMeterBillingPeriodForm.EndDateError =
						"End date must be greater or equal to begin date."
					formError = true
					mmBillingPeriodIndex++
					continue
				}
				calcBreakPoints = append(
					calcBreakPoints,
					[3]time.Time{
						endTime.AddDate(0, 0, -dayDiff),
						endTime.Time,
						endTime.AddDate(0, 0, dayDiff),
					},
				)
				// Shift begin time one day back,
				// so that eg. January end date minus begin date is 31 days.
				shiftedBeginTime := beginTime.AddDate(0, 0, -1)
				minTime := shiftedBeginTime.AddDate(0, 0, -dayDiff)
				calcBreakPoints = append(
					calcBreakPoints,
					[3]time.Time{
						minTime,
						shiftedBeginTime,
						beginTime.AddDate(0, 0, dayDiff),
					},
				)
				mainMeterBillingPeriod.EndDate = pgtype.Date{
					Time: endTime.Time, Valid: true}
				mainMeterBillingPeriod.BeginReadingValue =
					float64(beginReadingVal)
				mainMeterBillingPeriod.EndReadingValue = float64(endReadingVal)
				energyConsumption :=
					float64(endReadingVal) - float64(beginReadingVal)
				mainMeterBillingPeriod.EnergyConsumption = energyConsumption
				mainMeterBilling.EnergyConsumption += energyConsumption
				mainMeterBillingPeriod.ConsumedEnergyPrice =
					float64(consumedEnergyPrice)
				mainMeterBillingPeriod.ServicePrice = pgtype.Float8{
					Float64: servicePrice.Float64, Valid: servicePrice.Valid}
				totalPrice := float64(consumedEnergyPrice)
				if servicePrice.Valid {
					totalPrice += servicePrice.Float64
					mainMeterBilling.ServicePrice.Float64 +=
						servicePrice.Float64
					mainMeterBilling.ServicePrice.Valid = true
				}
				mainMeterBillingPeriod.TotalPrice = totalPrice
				if i == 0 {
					mainMeterBilling.BeginDate = pgtype.Date{
						Time: beginTime.Time, Valid: true}
					mainMeterBillingMinTime = minTime
				}
				if i == billingPeriodsLastIndex {
					mainMeterBilling.EndDate = pgtype.Date{
						Time: endTime.Time, Valid: true}
					mainMeterBillingMaxTime = endTime.Time
				}
				mainMeterBilling.ConsumedEnergyPrice += float64(
					consumedEnergyPrice)
				mainMeterBilling.TotalPrice += totalPrice
			}
		}
		mmBillingPeriodIndex++
	}
	slices.Reverse(tmplData.MainMeterBillingPeriods)
	if addBillingPeriod {
		tmplData.MainMeterBillingPeriods = append(
			tmplData.MainMeterBillingPeriods, &MainMeterBillingPeriodFormData{})
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	} else if removeBillingPeriod {
		if billingPeriodsLen > 1 {
			tmplData.MainMeterBillingPeriods =
				tmplData.MainMeterBillingPeriods[:billingPeriodsLastIndex]
		}
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	tx, err := s.postgresClient.Begin(ctx)
	if err != nil {
		slog.Error("error beginning transaction", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	subMeterReadings, err := qtx.GetSubMeterReadings(
		ctx, spinusdb.GetSubMeterReadingsParams{
			FkMainMeter: mainMeterID,
			DateMin:     pgtype.Date{Time: mainMeterBillingMinTime, Valid: true},
			DateMax:     pgtype.Date{Time: mainMeterBillingMaxTime, Valid: true},
		})
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	readingsLen := len(subMeterReadings)
	if readingsLen == 0 {
		tmplData.GeneralError = "There is no sub meter."
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	slog.Debug("billing", "calcBreakPoints", calcBreakPoints)
	calcBreakPointsLen := len(calcBreakPoints)
	breakPointReadings := make(map[time.Time]map[int32]*Reading)
	laterReadings := make(map[int32]*Reading)
	breakPointLastIndexes := make(map[int32]int)
	var additionalBreakPoints BreakPoints              // From latest to earliest.
	for _, subMeterReading := range subMeterReadings { // From latest to earliest.
		readingDate := subMeterReading.ReadingDate
		subMeterID := subMeterReading.SubMeterID
		readingVal := subMeterReading.ReadingValue.Float64
		readingTime := readingDate.Time
		readingValid := readingDate.Valid
		reading := &Reading{
			Value: readingVal,
			Time:  readingTime,
			Valid: readingValid,
		}
		lastIndex := breakPointLastIndexes[subMeterID]
		if lastIndex > calcBreakPointsLen {
			continue
		}
		if !readingValid { // Earliest reading.
			laterReading, ok := laterReadings[subMeterID]
			if ok {
				// Calculate valid readings.
				// Set invalid reading for break points without reading.
				lrTime := laterReading.Time
				var lrInBp bool // Later reading in break point range.
				var hasValidReading bool
				for _, bp := range calcBreakPoints {
					bpMin := bp[0]
					bpActual := bp[1]
					bpMax := bp[2]
					_, ok := breakPointReadings[bpActual]
					if !ok {
						breakPointReadings[bpActual] =
							make(map[int32]*Reading)
					}
					r, ok := breakPointReadings[bpActual][subMeterID]
					if ok {
						if r.Valid {
							hasValidReading = true
						}
					} else {
						breakPointReadings[bpActual][subMeterID] =
							&Reading{}
					}
					if !lrInBp && lrTime.Compare(bpMax) <= 0 &&
						lrTime.Compare(bpMin) >= 0 {
						// Later reading is in break point range.
						lrInBp = true
					}
				}
				// Sub meter has at least one valid break point reading.
				if hasValidReading {
					bp := [3]time.Time{
						lrTime.AddDate(0, 0, -dayDiff),
						lrTime,
						lrTime.AddDate(0, 0, dayDiff),
					}
					if !lrInBp &&
						!slices.Contains(additionalBreakPoints, bp) &&
						lrTime.Compare(mainMeterBillingMaxTime) <= 0 &&
						lrTime.Compare(mainMeterBillingMinTime) >= 0 {
						// Add later reading to additional break points.
						additionalBreakPoints = append(
							additionalBreakPoints, bp)
					}
				}
			} else {
				// No later reading, set invalid reading for all break points.
				for _, bp := range calcBreakPoints {
					bpActual := bp[1]
					_, ok := breakPointReadings[bpActual]
					if !ok {
						breakPointReadings[bpActual] =
							make(map[int32]*Reading)
					}
					breakPointReadings[bpActual][subMeterID] = &Reading{}
				}
			}
			continue // There is no other reading for this sub meter.
		}
		var additionalBp [3]time.Time
		var lrValGE bool // Later value greater or equal to current reading value.
		var lrVal float64
		lr, ok := laterReadings[subMeterID]
		if ok {
			lrVal = lr.Value
			if lrVal >= readingVal {
				// Later reading value is greater or equal to current reading
				// value.
				lrValGE = true
			} else {
				// Later reading value is lower than current reading value.
				lrTime := lr.Time
				bp := [3]time.Time{
					lrTime.AddDate(0, 0, -dayDiff),
					lrTime,
					lrTime.AddDate(0, 0, dayDiff),
				}
				if !slices.Contains(calcBreakPoints, bp) &&
					!slices.Contains(additionalBreakPoints, bp) &&
					lrTime.Compare(mainMeterBillingMaxTime) <= 0 &&
					lrTime.Compare(mainMeterBillingMinTime) >= 0 {
					// Add later reading to additional break points.
					additionalBreakPoints = append(additionalBreakPoints, bp)
				}
				bp = [3]time.Time{
					readingTime.AddDate(0, 0, -dayDiff),
					readingTime,
					readingTime.AddDate(0, 0, dayDiff),
				}
				if !slices.Contains(calcBreakPoints, bp) &&
					!slices.Contains(additionalBreakPoints, bp) &&
					readingTime.Compare(mainMeterBillingMaxTime) <= 0 &&
					readingTime.Compare(mainMeterBillingMinTime) >= 0 {
					// Add reading to additional break points.
					additionalBreakPoints = append(additionalBreakPoints, bp)
				}
			}
		} else {
			// No later reading.
			bp := [3]time.Time{
				readingTime.AddDate(0, 0, -dayDiff),
				readingTime,
				readingTime.AddDate(0, 0, dayDiff),
			}
			if !slices.Contains(calcBreakPoints, bp) &&
				!slices.Contains(additionalBreakPoints, bp) &&
				readingTime.Compare(mainMeterBillingMaxTime) <= 0 &&
				readingTime.Compare(mainMeterBillingMinTime) >= 0 {
				// Prepare current reading as possible additional break point.
				additionalBp = bp
			}
		}
		var readingInBp bool // Reading in break point range.
		for i := lastIndex; i < calcBreakPointsLen; i++ {
			bp := calcBreakPoints[i]
			bpMin := bp[0]
			bpActual := bp[1]
			bpMax := bp[2]
			_, ok := breakPointReadings[bpActual]
			if !ok {
				breakPointReadings[bpActual] = make(map[int32]*Reading)
			}
			prevBpReading, prevBpReadingOk := breakPointReadings[bpActual][subMeterID]
			if readingTime.After(bpMax) {
				// After break point max.
				if !readingInBp && !additionalBp[1].IsZero() {
					// Add additional break point when set.
					additionalBreakPoints = append(
						additionalBreakPoints, additionalBp)
				}
				break
			} else if readingTime.Compare(bpMin) >= 0 {
				// Between break point min and max.
				if !prevBpReadingOk || bpActual.Sub(readingTime) <=
					bpActual.Sub(prevBpReading.Time) {
					// Lower time difference or no previous reading.
					breakPointReadings[bpActual][subMeterID] = reading
				}
				if bpActual.Compare(readingTime) >= 0 {
					// No better reading possible.
					breakPointLastIndexes[subMeterID] += 1
				}
				readingInBp = true
			} else {
				// Before break point min.
				if lrValGE {
					readingDayDiff := lr.Time.Sub(readingTime).Hours() / 24
					newValPerDay := (lrVal - readingVal) / readingDayDiff
					newVal := readingVal +
						(newValPerDay * bpActual.Sub(readingTime).Hours() /
							24)
					newReading := &Reading{
						Value: newVal,
						Time:  bpActual,
						Valid: true,
					}
					breakPointReadings[bpActual][subMeterID] = newReading
					breakPointLastIndexes[subMeterID] += 1
				} else {
					// Set invalid reading for break point.
					breakPointReadings[bpActual][subMeterID] = &Reading{}
					breakPointLastIndexes[subMeterID] += 1
					// Possible additional break point is set already.
				}
			}
		}
		laterReadings[subMeterID] = reading
	}

	slog.Debug("billing", "additionalBreakPoints", additionalBreakPoints)
	additionalBreakPointsLen := len(additionalBreakPoints)
	if additionalBreakPointsLen > 0 {
		sort.Sort(sort.Reverse(additionalBreakPoints))
		laterReadings = make(map[int32]*Reading)
		breakPointLastIndexes = make(map[int32]int)        // From latest to earliest.
		for _, subMeterReading := range subMeterReadings { // From latest to earliest.
			readingDate := subMeterReading.ReadingDate
			subMeterID := subMeterReading.SubMeterID
			readingVal := subMeterReading.ReadingValue.Float64
			readingTime := readingDate.Time
			readingValid := readingDate.Valid
			reading := &Reading{
				Value: readingVal,
				Time:  readingTime,
				Valid: readingValid,
			}
			lastIndex := breakPointLastIndexes[subMeterID]
			if lastIndex > additionalBreakPointsLen {
				continue
			}
			if !readingValid { // Earliest reading.
				// Set invalid reading for break points without reading.
				for i := lastIndex; i < additionalBreakPointsLen; i++ {
					bp := additionalBreakPoints[i]
					bpActual := bp[1]
					_, ok := breakPointReadings[bpActual]
					if !ok {
						breakPointReadings[bpActual] =
							make(map[int32]*Reading)
					}
					_, ok = breakPointReadings[bpActual][subMeterID]
					if !ok {
						breakPointReadings[bpActual][subMeterID] =
							&Reading{}
					}
				}
				continue
			}
			var lrValGE bool
			var lrVal float64
			lr, ok := laterReadings[subMeterID]
			if ok {
				lrVal = lr.Value
				if lrVal >= readingVal {
					// Later reading value is greater or equal to current
					// reading value.
					lrValGE = true
				}
			}
			for i := lastIndex; i < additionalBreakPointsLen; i++ {
				bp := additionalBreakPoints[i]
				bpMin := bp[0]
				bpActual := bp[1]
				bpMax := bp[2]
				_, ok := breakPointReadings[bpActual]
				if !ok {
					breakPointReadings[bpActual] = make(map[int32]*Reading)
				}
				prevBpReading, prevBpReadingOk :=
					breakPointReadings[bpActual][subMeterID]
				if readingTime.After(bpMax) {
					// After break point max.
					break
				} else if readingTime.Compare(bpMin) >= 0 {
					// Between break point min and max.
					if !prevBpReadingOk || bpActual.Sub(readingTime) <=
						bpActual.Sub(prevBpReading.Time) {
						// Lower time difference or no previous reading.
						breakPointReadings[bpActual][subMeterID] = reading
					}
					if bpActual.Compare(readingTime) >= 0 {
						// No better reading possible.
						breakPointLastIndexes[subMeterID] += 1
					}
				} else {
					// Before break point min.
					if lrValGE {
						readingDayDiff :=
							lr.Time.Sub(readingTime).Hours() / 24
						newValPerDay :=
							(lrVal - readingVal) / readingDayDiff
						newVal := readingVal + (newValPerDay *
							bpActual.Sub(readingTime).Hours() / 24)
						newReading := &Reading{
							Value: newVal,
							Time:  bpActual,
							Valid: true,
						}
						breakPointReadings[bpActual][subMeterID] =
							newReading
						breakPointLastIndexes[subMeterID] += 1
					} else {
						// Set invalid reading for break point.
						breakPointReadings[bpActual][subMeterID] =
							&Reading{}
						breakPointLastIndexes[subMeterID] += 1
					}
				}
			}
			laterReadings[subMeterID] = reading
		}
		// Merge all break points.
		calcBreakPoints = append(calcBreakPoints, additionalBreakPoints...)
		sort.Sort(sort.Reverse(calcBreakPoints))
	}

	subMeterList, err := qtx.ListSubMeters(ctx, mainMeterID)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	subMeters := make(map[int32]spinusdb.ListSubMetersRow)
	for _, sm := range subMeterList {
		subMeter := sm
		subMeters[sm.ID] = subMeter
	}

	subMeterBillings := make(map[int32]*spinusdb.CreateSubMeterBillingParams)
	subMeterBillingPeriods := make(
		map[int]map[int32]*spinusdb.CreateSubMeterBillingPeriodParams)

	calcBreakPointsLen = len(calcBreakPoints)

	calcBpIndex := 0
	bp := calcBreakPoints[calcBpIndex]
	laterBreakPointReadings := breakPointReadings[bp[1]]
	calcBpIndex++

	subMeterLen := len(breakPointReadings[calcBreakPoints[calcBpIndex][1]])

	// Prepare main meter billing period.
	var mmServicePriceValid bool
	var mmServicePricePerSm float64
	mmBillingPeriodIndex = 0
	subMeterBillingPeriods[mmBillingPeriodIndex] = make(
		map[int32]*spinusdb.CreateSubMeterBillingPeriodParams)
	mmBillingPeriod := mainMeterBillingPeriods[mmBillingPeriodIndex]
	mmBeginTime := mmBillingPeriod.BeginDate.Time
	mmMinTime := mmBeginTime.AddDate(0, 0, -1) // Shifted one day back.
	mmEndTime := mmBillingPeriod.EndDate.Time
	mmDays := mmEndTime.Sub(mmMinTime).Hours() / 24
	mmBeginVal := mmBillingPeriod.BeginReadingValue
	mmEndVal := mmBillingPeriod.EndReadingValue
	mmConsumption := mmEndVal - mmBeginVal
	mmValPerDay := (mmEndVal - mmBeginVal) / mmDays
	mmConsumedEnergyPricePerUnit := mmBillingPeriod.ConsumedEnergyPrice / mmConsumption
	if mmBillingPeriod.ServicePrice.Valid {
		mmServicePriceValid = true
		mmServicePricePerSm = mmBillingPeriod.ServicePrice.Float64 / float64(subMeterLen)
	}
	mmLaterBpVal := mmEndVal // Main meter later break point value.

	for ; calcBpIndex < calcBreakPointsLen; calcBpIndex++ { // From latest to earliest.
		bp := calcBreakPoints[calcBpIndex]
		bpActual := bp[1]
		var mmBpVal float64
		var mmBpConsumption float64
		if bpActual.After(mmMinTime) {
			// Actual is between main meter billing period min and max.
			// Need to calculate main meter reading value.
			mmBpVal = mmBeginVal + (mmValPerDay * bpActual.Sub(mmMinTime).Hours() / 24)
		} else {
			mmBpVal = mmBeginVal
		}
		// Calculate main meter break point consumption.
		mmBpConsumption = mmLaterBpVal - mmBpVal
		mmLaterBpVal = mmBpVal
		var invalidCount int
		var readingsSum, laterReadingsSum float64
		readings := breakPointReadings[bpActual]
		for subMeterID, reading := range readings {
			// Calculate invalid count and sums for current and later break point.
			// Sum must be calculated every time even for later break point
			// because later reading can be lower then current reading.
			laterReading := laterBreakPointReadings[subMeterID]
			if !reading.Valid || !laterReading.Valid {
				invalidCount++
				continue
			}
			readingVal := reading.Value
			laterReadingVal := laterReading.Value
			if readingVal > laterReadingVal {
				invalidCount++
			} else {
				readingsSum += readingVal
				laterReadingsSum += laterReadingVal
			}

		}
		var energyConsumptionValidAddendum float64
		var energyConsumptionInvalidAddendum float64
		if invalidCount == 0 {
			// All valid, split difference equally to all sub meters.
			energyConsumptionValidAddendum =
				(mmBpConsumption - (laterReadingsSum - readingsSum)) /
					float64(subMeterLen)
		} else {
			// At least one invalid, split difference only to sub meters with invalid
			// reading.
			energyConsumptionInvalidAddendum =
				(mmBpConsumption - (laterReadingsSum - readingsSum)) /
					float64(invalidCount)
		}
		for subMeterID, reading := range readings {
			// Calculate energy consumption and energy consumption price.
			smBillingPeriod, ok :=
				subMeterBillingPeriods[mmBillingPeriodIndex][subMeterID]
			if !ok {
				smBillingPeriod = &spinusdb.CreateSubMeterBillingPeriodParams{}
				if mmServicePriceValid {
					smBillingPeriod.ServicePrice = pgtype.Float8{
						Float64: mmServicePricePerSm, Valid: true}
				}
				subMeterBillingPeriods[mmBillingPeriodIndex][subMeterID] =
					smBillingPeriod
			}
			var energyConsumption float64
			laterReading := laterBreakPointReadings[subMeterID]
			if !reading.Valid || !laterReading.Valid {
				energyConsumption = energyConsumptionInvalidAddendum
			} else {
				readingVal := reading.Value
				laterReadingVal := laterReading.Value
				if readingVal > laterReadingVal {
					energyConsumption = energyConsumptionInvalidAddendum
				} else {
					energyConsumption = laterReadingVal - readingVal +
						energyConsumptionValidAddendum
				}
			}
			smBillingPeriod.EnergyConsumption += energyConsumption
			smBillingPeriod.ConsumedEnergyPrice +=
				energyConsumption * mmConsumedEnergyPricePerUnit
		}
		if bpActual == mmMinTime {
			// Earliest break point for current main meter billing period.
			smBillingPeriods := subMeterBillingPeriods[mmBillingPeriodIndex]
			for smID, smBillingPeriod := range smBillingPeriods {
				// Calculate all prices for sub meter billing periods and main
				// meter billing period.
				smBilling, ok := subMeterBillings[smID]
				var smForm *MainMeterBillingSubMeterFormData
				if !ok {
					smBilling = &spinusdb.CreateSubMeterBillingParams{
						FkSubMeter: smID}
					subMeterBillings[smID] = smBilling
					subMeter := subMeters[smID]
					smForm = &MainMeterBillingSubMeterFormData{
						ID: smID,
						Subid: subMeter.Subid,
						MeterID: subMeter.MeterID,
						Email: subMeter.Email,
					}
					tmplData.SubMeterBillings = append(
						tmplData.SubMeterBillings, smForm)
				}

				energyConsumption := smBillingPeriod.EnergyConsumption
				consumedEnergyPrice := smBillingPeriod.ConsumedEnergyPrice
				var servicePrice float64
				if mmServicePriceValid {
					servicePrice = smBillingPeriod.ServicePrice.Float64
					smBilling.ServicePrice.Float64 += servicePrice
					smBilling.ServicePrice.Valid = true
					smForm.ServicePrice.Float64 += servicePrice
					smForm.ServicePrice.Valid = true
				}
				advancePrice := consumedEnergyPrice + servicePrice
				totalPrice := consumedEnergyPrice + servicePrice + advancePrice // TODO - minus previous advance price
				smBillingPeriod.AdvancePrice = advancePrice
				smBillingPeriod.TotalPrice = totalPrice
				smBilling.EnergyConsumption += energyConsumption
				smBilling.ConsumedEnergyPrice += consumedEnergyPrice
				smBilling.AdvancePrice += advancePrice
				smBilling.TotalPrice += totalPrice
				smForm.EnergyConsumption += energyConsumption
				smForm.ConsumedEnergyPrice += consumedEnergyPrice
				smForm.AdvancePrice += advancePrice
				smForm.TotalPrice += totalPrice
				mmBillingPeriod.AdvancePrice += advancePrice
				mmBillingPeriod.TotalPrice += advancePrice
			}
			// Update main meter billing.
			mmBillingPeriodAdvancePrice := mmBillingPeriod.AdvancePrice
			mainMeterBilling.AdvancePrice += mmBillingPeriodAdvancePrice
			mainMeterBilling.TotalPrice += mmBillingPeriodAdvancePrice
			mmBillingPeriodIndex++
			if mmBillingPeriodIndex == billingPeriodsLen {
				break
			}

			// Prepare main meter billing period.
			subMeterBillingPeriods[mmBillingPeriodIndex] = make(
				map[int32]*spinusdb.CreateSubMeterBillingPeriodParams)
			mmBillingPeriod = mainMeterBillingPeriods[mmBillingPeriodIndex]
			mmBeginTime = mmBillingPeriod.BeginDate.Time
			mmMinTime = mmBeginTime.AddDate(0, 0, -1)
			mmEndTime = mmBillingPeriod.EndDate.Time
			mmDays = mmEndTime.Sub(mmMinTime).Hours() / 24
			mmBeginVal = mmBillingPeriod.BeginReadingValue
			mmEndVal = mmBillingPeriod.EndReadingValue
			mmConsumption = mmEndVal - mmBeginVal
			mmValPerDay = (mmEndVal - mmBeginVal) / mmDays
			mmConsumedEnergyPricePerUnit =
				mmBillingPeriod.ConsumedEnergyPrice / mmConsumption
			mmServicePricePerSm =
				mmBillingPeriod.ServicePrice.Float64 / float64(subMeterLen)
			if mmBillingPeriod.ServicePrice.Valid {
				mmServicePriceValid = true
				mmServicePricePerSm =
					mmBillingPeriod.ServicePrice.Float64 / float64(subMeterLen)
			}
			mmLaterBpVal = mmEndVal
		}
		laterBreakPointReadings = breakPointReadings[bpActual]
	}

	// Calculate billing, do not create.
	if r.PostFormValue("calculate-billing") != "" {
		tmplData.Calculated = true
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}
	createdMainMeterBilling, err := qtx.CreateMainMeterBilling(ctx, mainMeterBilling)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	createdMainMeterBillingID := createdMainMeterBilling.ID

	createdSubMeterBillingIDs := make(map[int32]int32)

	for smID, smBilling := range subMeterBillings {
		smBilling.FkMainBilling = createdMainMeterBillingID
		createdSubMeterBilling, err := qtx.CreateSubMeterBilling(ctx, *smBilling)
		if err != nil {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		createdSubMeterBillingIDs[smID] = createdSubMeterBilling.ID
	}

	for i, mmBillingPeriod := range mainMeterBillingPeriods {
		mmBillingPeriod.FkMainBilling = createdMainMeterBillingID
		createdMainMeterBillingPeriod, err := qtx.CreateMainMeterBillingPeriod(
			ctx, *mmBillingPeriod)
		if err != nil {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		createdMainMeterBillingPeriodID := createdMainMeterBillingPeriod.ID
		smBillingPeriods := subMeterBillingPeriods[i]
		for smID, smBillingPeriod := range smBillingPeriods {
			smBillingPeriod.FkSubBilling = createdSubMeterBillingIDs[smID]
			smBillingPeriod.FkMainBillingPeriod = createdMainMeterBillingPeriodID
			_, err := qtx.CreateSubMeterBillingPeriod(ctx, *smBillingPeriod)
			if err != nil {
				slog.Error("error executing query", "err", err)
				s.HandleInternalServerError(w, r, err)
				return
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("error committing transaction", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	// TODO - redirect
}
