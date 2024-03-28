package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

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
	if r.Context().Value(userIDKey) != emptyUserIDValue {
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

	formData := SignUpFormData{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		formError = true
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

	email := r.PostFormValue("email")
	formData.Email = email
	parsedEmail, err := parseEmail(email)
	if err == nil {
		_, err := s.queries.GetUserByEmail(ctx, parsedEmail)
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

	parsedPassword, passwordErr := parsePassword(r.PostFormValue("password"))
	if passwordErr != nil {
		formData.PasswordError = passwordErr.Error()
		formError = true
	}
	parsedRepeatPassword, repeatPasswordErr := parsePassword(
		r.PostFormValue("repeat-password"))
	if repeatPasswordErr != nil {
		formData.RepeatPasswordError = repeatPasswordErr.Error()
		formError = true
	}
	if passwordErr == nil &&
		repeatPasswordErr == nil &&
		parsedPassword != parsedRepeatPassword {

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
		slog.Info("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		formError = true
		s.templates.Render(w, tmplName, formData)
		return
	}

	username := r.PostFormValue("username")
	formData.Username = username
	parsedUsername, err := parseUsername(username)
	if err != nil {
		formData.UsernameError = err.Error()
		formError = true
	}

	password := r.PostFormValue("password")
	formData.Password = password
	parsedPassword, err := parsePassword(password)
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
		ctx, spinusdb.GetUserParams{Username: parsedUsername, Crypt: parsedPassword})
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
	if ok == false {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeters, err := s.queries.ListMainMeters(r.Context(), userID)
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
		slog.Info("error parsing form", "err", err)
		formData.GeneralError = "Bad request"
		formError = true
		s.templates.Render(w, tmplName, formData)
		return
	}

	meterID := r.PostFormValue("meter-identification")
	formData.MeterID = meterID
	parsedMeterID, err := parseMainMeterID(meterID)
	if err != nil {
		formData.MeterIDError = err.Error()
		formError = true
	}

	energy := r.PostFormValue("energy")
	formData.Energy = energy
	parsedEnergy, err := parseEnergy(energy)
	if err != nil {
		formData.EnergyError = err.Error()
		formError = true
	}

	address := r.PostFormValue("address")
	formData.Address = address
	parsedAddress, err := parseAddress(address)
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
	if ok == false {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, err := s.queries.CreateMainMeter(
		ctx,
		spinusdb.CreateMainMeterParams{
			MeterID: parsedMeterID,
			Energy:  parsedEnergy,
			Address: parsedAddress,
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
	if ok == false {
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
	if ok == false {
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
	if ok == false {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mainMeter, ok := GetMainMeter(ctx)
	if ok == false {
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
		slog.Info("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		formError = true
		s.templates.Render(w, tmplName, tmplData)
		return
	}

	meterID := r.PostFormValue("meter-identification")
	tmplData.MeterID = meterID
	parsedMeterID, err := parseSubMeterID(meterID)
	if err != nil {
		tmplData.MeterIDError = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSubMeter(
		ctx,
		spinusdb.CreateSubMeterParams{
			FkMainMeter: mainMeter.ID,
			MeterID:     parsedMeterID,
			FkUser:      userID,
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
	if ok == false {
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
	if ok == false {
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
	if ok == false {
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
	if ok == false {
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
	if ok == false {
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
		slog.Info("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		formError = true
		s.templates.Render(w, tmplName, tmplData)
		return
	}

	readingValue := r.PostFormValue("reading-value")
	tmplData.ReadingValue = readingValue
	parsedReadingValue, err := parseReadingValue(readingValue)
	if err != nil {
		tmplData.ReadingValueError = err.Error()
		formError = true
	}

	subMeterID := subMeter.ID
	readingDate := r.PostFormValue("reading-date")
	tmplData.ReadingDate = readingDate
	parsedReadingDate, err := parseDate(readingDate)
	if err == nil {
		_, err = s.queries.GetSubMeterReadingForDate(
			ctx,
			spinusdb.GetSubMeterReadingForDateParams{
				FkSubMeter:  subMeterID,
				ReadingDate: parsedReadingDate,
			})
		if err == nil {
			tmplData.ReadingDateError = "Reading for the given date already exists."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	}
	if err != nil {
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
			ReadingValue: parsedReadingValue,
			ReadingDate:  parsedReadingDate,
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
	if ok == false {
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
	if ok == false {
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
	if ok == false {
		slog.Error("error getting main meter", "mainMeter", mainMeter)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	mainMeterID := mainMeter.ID

	var mainMeterbillingPeriodForms []*MainMeterBillingPeriodFormData
	tmplData := MainMeterBillingCreateTmplData{
		MainMeterBillingFormData: MainMeterBillingFormData{
			BillingPeriods: mainMeterbillingPeriodForms},
		Upper: MainMeterTmplData{ID: mainMeterID},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralError = "Bad request"
		formError = true
		s.templates.Render(w, tmplName, tmplData)
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
	if addBillingPeriod == false && removeBillingPeriod == false {
		parse = true
	}

	maxDayDiff := r.PostFormValue("max-day-diff")
	tmplData.MaxDayDiff = maxDayDiff
	parsedMaxDayDiff, err := parseMaxDayDiff(maxDayDiff)
	// TODO - use parsedMaxDayDiff
	if err != nil {
		tmplData.MaxDayDiffError = err.Error()
		formError = true
	}

	var mainMeterBillingPeriods []*spinusdb.CreateMainMeterBillingPeriodParams
	var breakPoints [][3]time.Time

	beginDates := r.PostForm["begin-date"]
	endDates := r.PostForm["end-date"]
	beginReadingValues := r.PostForm["begin-reading-value"]
	endReadingValues := r.PostForm["end-reading-value"]
	consumedEnergyPrices := r.PostForm["consumed-energy-price"]
	servicePrices := r.PostForm["service-price"]
	billingPeriodsLen := len(beginDates)
	for i := 0; i < billingPeriodsLen; i++ {
		mainMeterBillingPeriodForm := &MainMeterBillingPeriodFormData{}
		tmplData.BillingPeriods = append(
			tmplData.BillingPeriods, mainMeterBillingPeriodForm)
		beginDate := beginDates[i]
		mainMeterBillingPeriodForm.BeginDate = beginDate
		endDate := endDates[i]
		mainMeterBillingPeriodForm.EndDate = endDate
		beginReadingValue := beginReadingValues[i]
		mainMeterBillingPeriodForm.BeginReadingValue = beginReadingValue
		endReadingValue := endReadingValues[i]
		mainMeterBillingPeriodForm.EndReadingValue = endReadingValue
		consumedEnergyPrice := consumedEnergyPrices[i]
		mainMeterBillingPeriodForm.ConsumedEnergyPrice = consumedEnergyPrice
		servicePrice := servicePrices[i]
		mainMeterBillingPeriodForm.ServicePrice = servicePrice
		if parse {
			mainMeterBillingPeriod := &spinusdb.CreateMainMeterBillingPeriodParams{}
			mainMeterBillingPeriods = append(
				mainMeterBillingPeriods, mainMeterBillingPeriod)
			parsedBeginDate, err := parseDate(beginDate)
			if err == nil {
				if i > 0 {
					previousEndDate :=
						mainMeterBillingPeriods[i-1].EndDate
					if previousEndDate.Valid {
						if previousEndDate.Time != parsedBeginDate.Time {
							mainMeterBillingPeriodForm.BeginDateError =
								"Begin date must be the same as " +
									"previous billing " +
									"period's end date."
							formError = true
						} else if formError == false {
							mainMeterBillingPeriod.BeginDate =
								parsedBeginDate
						}

					}
				} else if formError == false {
					mainMeterBillingPeriod.BeginDate = parsedBeginDate
				}
			} else {
				mainMeterBillingPeriodForm.BeginDateError = err.Error()
				formError = true
			}
			parsedEndDate, err := parseDate(endDate)
			if err == nil {
				if formError == false {
					mainMeterBillingPeriod.EndDate = parsedEndDate
					if parsedBeginDate.Time.Compare(parsedEndDate.Time) <= 0 {
						mainMeterBillingPeriodForm.EndDateError =
							"End date must be greater than begin date."
						formError = true
					}
				}
			} else {
				mainMeterBillingPeriodForm.EndDateError = err.Error()
				formError = true
			}
			if formError == false {
				dayDiff := int(parsedMaxDayDiff.Int32)
				parsedBeginTime := parsedBeginDate.Time
				breakPoints = append(
					breakPoints,
					[3]time.Time{
						parsedBeginTime.AddDate(0, 0, -dayDiff),
						parsedBeginTime,
						parsedBeginTime.AddDate(0, 0, dayDiff),
					},
				)
				if i == billingPeriodsLen {
					parsedEndTime := parsedEndDate.Time
					breakPoints = append(
						breakPoints,
						[3]time.Time{
							parsedEndTime.AddDate(0, 0, -dayDiff),
							parsedEndTime,
							parsedEndTime.AddDate(0, 0, dayDiff),
						},
					)
				}
			}
			parsedBeginReadingValue, err := parseReadingValue(beginReadingValue)
			if err == nil {
				if formError == false {
					mainMeterBillingPeriod.BeginReadingValue =
						parsedBeginReadingValue
				}
			} else {
				mainMeterBillingPeriodForm.BeginReadingValueError = err.Error()
				formError = true
			}
			parsedEndReadingValue, err := parseReadingValue(endReadingValue)
			if err == nil {
				if formError == false {
					mainMeterBillingPeriod.EndReadingValue =
						parsedEndReadingValue
				}
			} else {
				mainMeterBillingPeriodForm.EndReadingValueError = err.Error()
				formError = true
			}
			parsedConsumedEnergyPrice, err := parseConsumedEnergyPrice(
				consumedEnergyPrice)
			if err == nil {
				if formError == false {
					mainMeterBillingPeriod.ConsumedEnergyPrice =
						parsedConsumedEnergyPrice
				}
			} else {
				mainMeterBillingPeriodForm.ConsumedEnergyPriceError = err.Error()
				formError = true
			}
			parsedServicePrice, err := parseServicePrice(servicePrice)
			if err == nil {
				if formError == false {
					mainMeterBillingPeriod.ServicePrice = parsedServicePrice
				}
			} else {
				mainMeterBillingPeriodForm.ServicePriceError = err.Error()
				formError = true
			}
		}
	}
	if addBillingPeriod {
		tmplData.BillingPeriods = append(
			tmplData.BillingPeriods, &MainMeterBillingPeriodFormData{})
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	} else if removeBillingPeriod {
		billingPeriodsLen := len(tmplData.BillingPeriods)
		if billingPeriodsLen > 1 {
			tmplData.BillingPeriods = tmplData.BillingPeriods[:billingPeriodsLen-1]
		}
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	// subMeterReadings, err := s.queries.GetSubMeterReadings(r.Context(), mainMeterID)
	// if err != nil {
	// 	slog.Error("error executing query", "err", err)
	// 	s.HandleInternalServerError(w, r, err)
	// 	return
	// }
	// readingsLen := len(subMeterReadings)
	// if readingsLen == 0 {
	// 	tmplData.GeneralError = "There is no sub meter."
	// 	s.renderTemplate(w, r, tmplName, tmplData)
	// 	return
	// }

	// billingPeriodsLen = len(mainMeterBillingPeriods)
	// if billingPeriodsLen == 0 {
	// 	tmplData.GeneralError = "No billing period provided."
	// 	s.renderTemplate(w, r, tmplName, tmplData)
	// 	return
	// }
	// var additionalBreakPoints []time.Time
	// breakPointsLen := len(breakPoints)
	// previousSubMeterReading := subMeterReadings[0]
	// breakPointReadings := make(map[time.Time]map[int32]*Reading)
	// for _, subMeterReading := range subMeterReadings {
	// 	readingDate := subMeterReading.ReadingDate
	// 	if readingDate.Valid == false {
	// 		previousSubMeterReading = subMeterReading
	// 		continue
	// 	}
	// 	subMeterID := subMeterReading.ID
	// 	readingTime := readingDate.Time
	// 	reading := &Reading{
	// 		Value: subMeterReading.ReadingValue.Float64,
	// 		Time:  readingTime,
	// 	}
	// 	for i, breakPoint := range breakPoints {
	// 		breakPointMin := breakPoint[0]
	// 		breakPointActual := breakPoint[1]
	// 		breakPointMax := breakPoint[2]
	// 		if readingTime.Before(breakPointMin) || readingTime.After(breakPointMax) {
	// 			continue
	// 		}
	// 		_, ok := breakPointReadings[breakPointActual]
	// 		if ok == false {
	// 			breakPointReadings[breakPointActual] = make(map[int32]*Reading)
	// 		}
	// 		previousReading, ok := breakPointReadings[breakPointActual][subMeterID]
	// 		if ok == false {
	// 			breakPointReadings[breakPointActual][subMeterID] = reading
	// 		} else if breakPointActual.Sub(readingTime) <=
	// 			breakPointActual.Sub(previousReading.Time) {

	// 			breakPointReadings[breakPointActual][subMeterID] = reading
	// 		}

	// 		if i == breakPointsLen &&
	// 			subMeterID != previousSubMeterReading.ID &&
	// 			previousSubMeterReading.ReadingDate.Valid {

	// 			additionalBreakPoints = append(
	// 				additionalBreakPoints, breakPointActual)
	// 		}
	// 	}
	// 	previousSubMeterReading = subMeterReading
	// }

	s.renderTemplate(w, r, tmplName, tmplData)
}
