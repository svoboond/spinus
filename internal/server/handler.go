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

func (s *Server) HandleInternalServerError(
	w http.ResponseWriter, r *http.Request, err error) {

	s.renderTemplate(w, r, errorTmplName, err.Error())
}

func (s *Server) renderTemplate(
	w http.ResponseWriter, r *http.Request, name string, data any) {

	const upperTmplName = "upper"

	var buf bytes.Buffer
	var userLoggedIn bool
	if r.Context().Value(userIDKey) != emptyUserIDVal {
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

	form := SignUpForm{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		form.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, form); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	ctx := r.Context()

	iUsername := r.PostFormValue("username")
	form.Username = iUsername
	username, err := parseUsername(iUsername)
	if err == nil {
		_, err := s.queries.GetUserByUsername(ctx, string(username))
		if err == nil {
			form.UsernameErr = "Username is already taken."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		form.UsernameErr = err.Error()
		formError = true
	}

	iEmail := r.PostFormValue("email")
	form.Email = iEmail
	email, err := parseEmail(iEmail)
	if err == nil {
		_, err := s.queries.GetUserByEmail(ctx, string(email))
		if err == nil {
			form.EmailErr = "Email is already assigned to another account."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		form.EmailErr = err.Error()
		formError = true
	}

	password, passwordErr := parsePassword(r.PostFormValue("password"))
	if passwordErr != nil {
		form.PasswordErr = passwordErr.Error()
		formError = true
	}
	repeatPassword, repeatPasswordErr := parsePassword(
		r.PostFormValue("repeat-password"))
	if repeatPasswordErr != nil {
		form.RepeatPasswordErr = repeatPasswordErr.Error()
		formError = true
	}
	if passwordErr == nil && repeatPasswordErr == nil && password != repeatPassword {
		form.PasswordErr = "Passwords do not match."
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, form)
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
		ctx = context.WithValue(ctx, userIDKey, emptyUserIDVal)
		s.HandleInternalServerError(w, r.WithContext(ctx), err)
		return
	}
	ctx = context.WithValue(ctx, userIDKey, emptyUserIDVal)
	s.renderTemplate(w, r.WithContext(ctx), tmplName, nil)
}

func (s *Server) HandlePostLogIn(w http.ResponseWriter, r *http.Request) {
	const tmplName = "logIn"
	form := LogInForm{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		form.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, form); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iUsername := r.PostFormValue("username")
	form.Username = iUsername
	username, err := parseUsername(iUsername)
	if err != nil {
		form.UsernameErr = err.Error()
		formError = true
	}

	iPassword := r.PostFormValue("password")
	form.Password = iPassword
	password, err := parsePassword(iPassword)
	if err != nil {
		form.PasswordErr = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, form)
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
			form.GeneralErr = "Wrong username or password."
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

func (s *Server) HandleGetMmList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmList"

	ctx := r.Context()
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mms, err := s.queries.ListUserMms(r.Context(), userID)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	s.renderTemplate(w, r, tmplName, mms)
}

func (s *Server) HandleGetMmCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmCreate"
	s.renderTemplate(w, r, tmplName, nil)
}

func (s *Server) HandlePostMmCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmCreate"
	form := MmForm{}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		form.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, form); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iMeterID := r.PostFormValue("meter-identification")
	form.MeterID = iMeterID
	meterID, err := parseMmID(iMeterID)
	if err != nil {
		form.MeterIDErr = err.Error()
		formError = true
	}

	iEnergy := r.PostFormValue("energy")
	form.Energy = iEnergy
	energy, err := parseEnergy(iEnergy)
	if err != nil {
		form.EnergyErr = err.Error()
		formError = true
	}

	iAddress := r.PostFormValue("address")
	form.Address = iAddress
	address, err := parseAddress(iAddress)
	if err != nil {
		form.AddressErr = err.Error()
		formError = true
	}

	iCurrencyCode := r.PostFormValue("currency-code")
	form.CurrencyCode = iCurrencyCode
	currencyCode, err := parseCurrencyCode(iCurrencyCode)
	if err != nil {
		form.CurrencyCodeErr = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, form)
		return
	}

	ctx := r.Context()
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mm, err := s.queries.CreateMm(
		ctx,
		spinusdb.CreateMmParams{
			MeterID:      string(meterID),
			Energy:       energy,
			Address:      string(address),
			CurrencyCode: string(currencyCode),
			FkUser:       userID,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r, fmt.Sprintf("/main-meter/%d/overview", mm.ID), http.StatusSeeOther)
}

func (s *Server) HandleGetMmOverview(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmOverview"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		MmOverviewTmpl{
			GetMmRow: mm,
			Upper:    MmTmpl{ID: mm.ID},
		},
	)
}

func (s *Server) HandleGetSmCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smCreate"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SmCreateTmpl{
			SmForm: SmForm{},
			Upper:  MmTmpl{ID: mm.ID},
		},
	)
}

func (s *Server) HandlePostSmCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smCreate"

	ctx := r.Context()
	userID, ok := UserID(ctx)
	if !ok {
		slog.Error("error getting user ID", "userID", userID)
		s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
		return
	}
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	tmplData := SmCreateTmpl{
		SmForm: SmForm{},
		Upper:  MmTmpl{ID: mm.ID},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iMeterID := r.PostFormValue("meter-identification")
	tmplData.MeterID = iMeterID
	smID, err := parseSmID(iMeterID)
	if err != nil {
		tmplData.MeterIDErr = err.Error()
		formError = true
	}

	IFinBalance := r.PostFormValue("fin-balance")
	tmplData.FinBalance = IFinBalance
	finBalance, err := parseFinBalance(IFinBalance)
	if err != nil {
		tmplData.FinBalanceErr = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSm(
		ctx,
		spinusdb.CreateSmParams{
			FkMm:       mm.ID,
			MeterID:    pgtype.Text{String: string(smID), Valid: true},
			FinBalance: float64(finBalance),
			FkUser:     userID,
		},
	)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w, r,
		fmt.Sprintf("/main-meter/%d/sub-meter/list", mm.ID), http.StatusSeeOther,
	)
}

func (s *Server) HandleGetSmList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smList"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	mmID := mm.ID
	sms, err := s.queries.ListSms(r.Context(), mmID)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	s.renderTemplate(
		w, r,
		tmplName,
		SmListTmpl{
			Sms:   sms,
			Upper: MmTmpl{ID: mmID},
		},
	)
}

func (s *Server) HandleGetSmOverview(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smOverview"

	ctx := r.Context()
	sm, ok := GetSm(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", sm)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}

	s.renderTemplate(
		w, r,
		tmplName,
		SmOverviewTmpl{
			GetSmRow: sm,
			Upper:    SmTmpl{MmID: sm.MmID, Subid: sm.Subid},
		},
	)
}

func (s *Server) HandleGetSmRdgList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smRdgList"

	ctx := r.Context()
	sm, ok := GetSm(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", sm)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}
	subid := sm.Subid
	smRdgs, err := s.queries.ListSmRdgs(r.Context(), subid)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SmRdgListTmpl{
			SmRdgs: smRdgs,
			Upper:  SmTmpl{MmID: sm.MmID, Subid: subid},
		},
	)
}

func (s *Server) HandleGetSmRdgCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smRdgCreate"

	ctx := r.Context()
	sm, ok := GetSm(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", sm)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		SmRdgCreateTmpl{
			SmRdgForm: SmRdgForm{},
			Upper:     SmTmpl{MmID: sm.MmID, Subid: sm.Subid},
		},
	)
}

func (s *Server) HandlePostSmRdgCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "smRdgCreate"

	ctx := r.Context()
	sm, ok := GetSm(ctx)
	if !ok {
		slog.Error("error getting sub meter", "subMeter", sm)
		s.HandleInternalServerError(w, r, errors.New("error getting sub meter"))
		return
	}

	mmID := sm.MmID
	subid := sm.Subid
	tmplData := SmRdgCreateTmpl{
		SmRdgForm: SmRdgForm{},
		Upper:     SmTmpl{MmID: mmID, Subid: subid},
	}
	var formError bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	iRdgVal := r.PostFormValue("rdg-val")
	tmplData.RdgVal = iRdgVal
	rdgVal, err := parseRdgVal(iRdgVal)
	if err != nil {
		tmplData.RdgValErr = err.Error()
		formError = true
	}

	smID := sm.ID
	iRdgDate := r.PostFormValue("rdg-date")
	tmplData.RdgDate = iRdgDate
	rdgTime, err := parseDate(iRdgDate)
	rdgDate := pgtype.Date{Time: rdgTime.Time, Valid: true}
	if err == nil {
		_, err = s.queries.GetSmRdgForDate(
			ctx, spinusdb.GetSmRdgForDateParams{FkSm: smID, RdgDate: rdgDate})
		if err == nil {
			tmplData.RdgDateErr = "Reading for the given date already exists."
			formError = true
		} else if err != pgx.ErrNoRows {
			s.HandleInternalServerError(w, r, err)
			return
		}
	} else {
		tmplData.RdgDateErr = err.Error()
		formError = true
	}

	if formError {
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	_, err = s.queries.CreateSmRdg(
		ctx,
		spinusdb.CreateSmRdgParams{
			FkSm:     sm.ID,
			RdgVal: float64(rdgVal),
			RdgDate:  rdgDate,
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
			"/main-meter/%d/sub-meter/%d/reading/list", mmID, subid),
		http.StatusSeeOther,
	)
}

func (s *Server) HandleGetMmBillList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmBillList"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	mmID := mm.ID
	bills, err := s.queries.ListMmBills(r.Context(), mmID)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	s.renderTemplate(
		w, r, tmplName,
		MmBillListTmpl{
			MmBills: bills,
			Upper:   MmTmpl{ID: mmID},
		},
	)
}

func (s *Server) HandleGetMmBillCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmBillCreate"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}
	s.renderTemplate(
		w, r,
		tmplName,
		MmBillCreateTmpl{
			MmBillForm: NewMmBillForm(),
			Upper:      MmTmpl{ID: mm.ID},
		},
	)
}

func (s *Server) HandlePostMmBillCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mmBillCreate"

	ctx := r.Context()
	mm, ok := GetMm(ctx)
	if !ok {
		slog.Error("error getting main meter", "mainMeter", mm)
		s.HandleInternalServerError(w, r, errors.New("error getting main meter"))
		return
	}

	mmID := mm.ID

	var mmBillPeriodForms []*MmBillPeriodForm
	var smBillForms SmBillForms
	smIDBillForms := make(map[int32]*SmBillForm)
	tmplData := MmBillCreateTmpl{
		MmBillForm: MmBillForm{
			MmBillPeriods: mmBillPeriodForms,
			SmBills:       smBillForms,
		},
		Upper: MmTmpl{ID: mmID},
	}
	var formErr bool
	if err := r.ParseForm(); err != nil {
		slog.Error("error parsing form", "err", err)
		tmplData.GeneralErr = "Bad request"
		if err := s.templates.Render(w, tmplName, tmplData); err != nil {
			slog.Error("error rendering template", "template", tmplName, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	var addBillPeriod bool
	if r.PostFormValue("add-bill-period") != "" {
		addBillPeriod = true
	}
	var removeBillPeriod bool
	if r.PostFormValue("remove-bill-period") != "" {
		removeBillPeriod = true
	}
	var parse bool
	if !addBillPeriod && !removeBillPeriod {
		parse = true
	}

	mmBill := spinusdb.CreateMmBillParams{FkMm: mmID}

	iMaxDayDiff := r.PostFormValue("max-day-diff")
	tmplData.MaxDayDiff = iMaxDayDiff
	maxDayDiff, err := parseMaxDayDiff(iMaxDayDiff)
	dayDiff := int(maxDayDiff)
	if err != nil {
		tmplData.MaxDayDiffErr = err.Error()
		formErr = true
	}
	mmBill.MaxDayDiff = int32(maxDayDiff)

	// From latest to earliest.
	var mmBillPeriods []*spinusdb.CreateMmBillPeriodParams
	var calcBPs BreakPoints // From latest to earliest.

	iBeginDates := r.PostForm["begin-date"]
	iEndDates := r.PostForm["end-date"]
	iBeginRdgVals := r.PostForm["begin-rdg-val"]
	iEndRdgVals := r.PostForm["end-rdg-val"]
	iConsumEnergyPrices := r.PostForm["consum-energy-price"]
	iServicePrices := r.PostForm["service-price"]

	billPeriodsLen := len(iBeginDates)
	if billPeriodsLen == 0 {
		tmplData.GeneralErr = "No billing period provided."
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}
	billPeriodsLastIndex := billPeriodsLen - 1
	var mmBillMinTime time.Time // Shifted one day back.
	var mmBillMaxTime time.Time

	mmBillPeriodIndex := 0
	for i := billPeriodsLastIndex; i >= 0; i-- {
		mmBillPeriodForm := &MmBillPeriodForm{}
		tmplData.MmBillPeriods = append(
			tmplData.MmBillPeriods, mmBillPeriodForm)
		iBeginDate := iBeginDates[i]
		mmBillPeriodForm.BeginDate = iBeginDate
		iEndDate := iEndDates[i]
		mmBillPeriodForm.EndDate = iEndDate
		iBeginRdgVal := iBeginRdgVals[i]
		mmBillPeriodForm.BeginRdgVal = iBeginRdgVal
		iEndRdgVal := iEndRdgVals[i]
		mmBillPeriodForm.EndRdgVal = iEndRdgVal
		iConsumEnergyPrice := iConsumEnergyPrices[i]
		mmBillPeriodForm.ConsumEnergyPrice = iConsumEnergyPrice
		iServicePrice := iServicePrices[i]
		mmBillPeriodForm.ServicePrice = iServicePrice
		if parse {
			mmBillPeriod := &spinusdb.CreateMmBillPeriodParams{}
			mmBillPeriods = append(
				mmBillPeriods, mmBillPeriod)
			beginTime, err := parseDate(iBeginDate)
			if err != nil {
				mmBillPeriodForm.BeginDateErr = err.Error()
				formErr = true
			}
			endTime, err := parseDate(iEndDate)
			if err != nil {
				mmBillPeriodForm.EndDateErr = err.Error()
				formErr = true
			}
			beginRdgVal, err := parseRdgVal(iBeginRdgVal)
			if err != nil {
				mmBillPeriodForm.BeginRdgValErr = err.Error()
				formErr = true
			}
			endRdgVal, err := parseRdgVal(iEndRdgVal)
			if err != nil {
				mmBillPeriodForm.EndRdgValErr = err.Error()
				formErr = true
			}
			consumEnergyPrice, err := parseConsumEnergyPrice(iConsumEnergyPrice)
			if err != nil {
				mmBillPeriodForm.ConsumEnergyPriceErr = err.Error()
				formErr = true
			}
			servicePrice, err := parseServicePrice(iServicePrice)
			if err != nil {
				mmBillPeriodForm.ServicePriceErr = err.Error()
				formErr = true
			}
			if !formErr {
				if i != billPeriodsLastIndex {
					laterIndex := mmBillPeriodIndex - 1
					previousBeginDate := mmBillPeriods[laterIndex].BeginDate
					if endTime.AddDate(0, 0, 1) != previousBeginDate.Time {
						laterBillPeriod := tmplData.MmBillPeriods[laterIndex]
						laterBillPeriod.BeginDateErr =
							"Begin date must follow previous billing period's end date."
						formErr = true
						mmBillPeriodIndex++
						continue
					}
				}
				mmBillPeriod.BeginDate = pgtype.Date{
					Time: beginTime.Time, Valid: true}
				if endTime.Before(beginTime.Time) {
					mmBillPeriodForm.EndDateErr =
						"End date must be greater or equal to begin date."
					formErr = true
					mmBillPeriodIndex++
					continue
				}
				calcBPs = append(
					calcBPs,
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
				calcBPs = append(
					calcBPs,
					[3]time.Time{
						minTime,
						shiftedBeginTime,
						beginTime.AddDate(0, 0, dayDiff),
					},
				)
				mmBillPeriod.EndDate = pgtype.Date{Time: endTime.Time, Valid: true}
				mmBillPeriod.BeginRdgVal = float64(beginRdgVal)
				mmBillPeriod.EndRdgVal = float64(endRdgVal)
				energyConsum := float64(endRdgVal) - float64(beginRdgVal)
				mmBillPeriod.EnergyConsum = energyConsum
				mmBill.EnergyConsum += energyConsum
				mmBillPeriod.ConsumEnergyPrice = float64(consumEnergyPrice)
				mmBillPeriod.ServicePrice = pgtype.Float8{
					Float64: servicePrice.Float64, Valid: servicePrice.Valid}
				totalPrice := float64(consumEnergyPrice)
				if servicePrice.Valid {
					totalPrice += servicePrice.Float64
					mmBill.ServicePrice.Float64 += servicePrice.Float64
					mmBill.ServicePrice.Valid = true
				}
				mmBillPeriod.TotalPrice = totalPrice
				if i == 0 {
					mmBill.BeginDate = pgtype.Date{Time: beginTime.Time, Valid: true}
					mmBillMinTime = minTime
				}
				if i == billPeriodsLastIndex {
					mmBill.EndDate = pgtype.Date{Time: endTime.Time, Valid: true}
					mmBillMaxTime = endTime.Time
				}
				mmBill.ConsumEnergyPrice += float64(
					consumEnergyPrice)
			}
		}
		mmBillPeriodIndex++
	}
	slices.Reverse(tmplData.MmBillPeriods)
	if addBillPeriod {
		tmplData.MmBillPeriods = append(tmplData.MmBillPeriods, &MmBillPeriodForm{})
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	} else if removeBillPeriod {
		if billPeriodsLen > 1 {
			tmplData.MmBillPeriods = tmplData.MmBillPeriods[:billPeriodsLastIndex]
		}
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	if formErr {
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

	smRdgs, err := qtx.GetSmRdgs(
		ctx, spinusdb.GetSmRdgsParams{
			FkMm:    mmID,
			DateMin: pgtype.Date{Time: mmBillMinTime, Valid: true},
			DateMax: pgtype.Date{Time: mmBillMaxTime, Valid: true},
		})
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	rdgLen := len(smRdgs)
	if rdgLen == 0 {
		tmplData.GeneralErr = "There is no sub meter."
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}

	slog.Debug("billing", "calculationBreakPoints", calcBPs)
	calcBPsLen := len(calcBPs)
	bpRdgs := make(map[time.Time]map[int32]*Rdg)
	laterRdgs := make(map[int32]*Rdg)
	bpLastIndexes := make(map[int32]int)
	var additionalBPs BreakPoints  // From latest to earliest.
	for _, smRdg := range smRdgs { // From latest to earliest.
		rdgDate := smRdg.RdgDate
		smID := smRdg.SmID
		rdgVal := smRdg.RdgVal.Float64
		rdgTime := rdgDate.Time
		rdgValid := rdgDate.Valid
		rdg := &Rdg{
			Value: rdgVal,
			Time:  rdgTime,
			Valid: rdgValid,
		}
		lastIndex := bpLastIndexes[smID]
		if lastIndex > calcBPsLen {
			continue
		}
		if !rdgValid { // Earliest reading.
			laterRdg, ok := laterRdgs[smID]
			if ok {
				// Calculate valid readings.
				// Set invalid reading for break points without reading.
				lrTime := laterRdg.Time
				var lrInBp bool // Later reading in break point range.
				var hasValidRdg bool
				for _, bp := range calcBPs {
					bpMin := bp[0]
					bpActual := bp[1]
					bpMax := bp[2]
					_, ok := bpRdgs[bpActual]
					if !ok {
						bpRdgs[bpActual] = make(map[int32]*Rdg)
					}
					r, ok := bpRdgs[bpActual][smID]
					if ok {
						if r.Valid {
							hasValidRdg = true
						}
					} else {
						bpRdgs[bpActual][smID] = &Rdg{}
					}
					if !lrInBp && lrTime.Compare(bpMax) <= 0 &&
						lrTime.Compare(bpMin) >= 0 {
						// Later reading is in break point range.
						lrInBp = true
					}
				}
				// Sub meter has at least one valid break point reading.
				if hasValidRdg {
					bp := [3]time.Time{
						lrTime.AddDate(0, 0, -dayDiff),
						lrTime,
						lrTime.AddDate(0, 0, dayDiff),
					}
					if !lrInBp &&
						!slices.Contains(additionalBPs, bp) &&
						lrTime.Compare(mmBillMaxTime) <= 0 &&
						lrTime.Compare(mmBillMinTime) >= 0 {
						// Add later reading to additional break points.
						additionalBPs = append(additionalBPs, bp)
					}
				}
			} else {
				// No later reading, set invalid reading for all break points.
				for _, bp := range calcBPs {
					bpActual := bp[1]
					_, ok := bpRdgs[bpActual]
					if !ok {
						bpRdgs[bpActual] = make(map[int32]*Rdg)
					}
					bpRdgs[bpActual][smID] = &Rdg{}
				}
			}
			continue // There is no other reading for this sub meter.
		}
		var additionalBp [3]time.Time
		var lrValGE bool // Later value greater or equal to current reading value.
		var lrVal float64
		lr, ok := laterRdgs[smID]
		if ok {
			lrVal = lr.Value
			if lrVal >= rdgVal {
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
				if !slices.Contains(calcBPs, bp) &&
					!slices.Contains(additionalBPs, bp) &&
					lrTime.Compare(mmBillMaxTime) <= 0 &&
					lrTime.Compare(mmBillMinTime) >= 0 {
					// Add later reading to additional break points.
					additionalBPs = append(additionalBPs, bp)
				}
				bp = [3]time.Time{
					rdgTime.AddDate(0, 0, -dayDiff),
					rdgTime,
					rdgTime.AddDate(0, 0, dayDiff),
				}
				if !slices.Contains(calcBPs, bp) &&
					!slices.Contains(additionalBPs, bp) &&
					rdgTime.Compare(mmBillMaxTime) <= 0 &&
					rdgTime.Compare(mmBillMinTime) >= 0 {
					// Add reading to additional break points.
					additionalBPs = append(additionalBPs, bp)
				}
			}
		} else {
			// No later reading.
			bp := [3]time.Time{
				rdgTime.AddDate(0, 0, -dayDiff),
				rdgTime,
				rdgTime.AddDate(0, 0, dayDiff),
			}
			if !slices.Contains(calcBPs, bp) &&
				!slices.Contains(additionalBPs, bp) &&
				rdgTime.Compare(mmBillMaxTime) <= 0 &&
				rdgTime.Compare(mmBillMinTime) >= 0 {
				// Prepare current reading as possible additional break point.
				additionalBp = bp
			}
		}
		var rdgInBp bool // Reading in break point range.
		for i := lastIndex; i < calcBPsLen; i++ {
			bp := calcBPs[i]
			bpMin := bp[0]
			bpActual := bp[1]
			bpMax := bp[2]
			_, ok := bpRdgs[bpActual]
			if !ok {
				bpRdgs[bpActual] = make(map[int32]*Rdg)
			}
			prevBpRdg, prevBpRdgOk := bpRdgs[bpActual][smID]
			if rdgTime.After(bpMax) {
				// After break point max.
				if !rdgInBp && !additionalBp[1].IsZero() {
					// Add additional break point when set.
					additionalBPs = append(additionalBPs, additionalBp)
				}
				break
			} else if rdgTime.Compare(bpMin) >= 0 {
				// Between break point min and max.
				if !prevBpRdgOk || bpActual.Sub(rdgTime) <=
					bpActual.Sub(prevBpRdg.Time) {
					// Lower time difference or no previous reading.
					bpRdgs[bpActual][smID] = rdg
				}
				if bpActual.Compare(rdgTime) >= 0 {
					// No better reading possible.
					bpLastIndexes[smID] += 1
				}
				rdgInBp = true
			} else {
				// Before break point min.
				if lrValGE {
					rdgDayDiff := lr.Time.Sub(rdgTime).Hours() / 24
					newValPerDay := (lrVal - rdgVal) / rdgDayDiff
					newVal :=
						rdgVal + (newValPerDay * bpActual.Sub(rdgTime).Hours() / 24)
					newRdg := &Rdg{
						Value: newVal,
						Time:  bpActual,
						Valid: true,
					}
					bpRdgs[bpActual][smID] = newRdg
					bpLastIndexes[smID] += 1
				} else {
					// Set invalid reading for break point.
					bpRdgs[bpActual][smID] = &Rdg{}
					bpLastIndexes[smID] += 1
					// Possible additional break point is set already.
				}
			}
		}
		laterRdgs[smID] = rdg
	}

	slog.Debug("billing", "additionalBreakPoints", additionalBPs)
	additionalBPsLen := len(additionalBPs)
	if additionalBPsLen > 0 {
		sort.Sort(sort.Reverse(additionalBPs))
		laterRdgs = make(map[int32]*Rdg)
		bpLastIndexes = make(map[int32]int) // From latest to earliest.
		for _, smRdg := range smRdgs {      // From latest to earliest.
			rdgDate := smRdg.RdgDate
			smID := smRdg.SmID
			rdgVal := smRdg.RdgVal.Float64
			rdgTime := rdgDate.Time
			rdgValid := rdgDate.Valid
			rdg := &Rdg{
				Value: rdgVal,
				Time:  rdgTime,
				Valid: rdgValid,
			}
			lastIndex := bpLastIndexes[smID]
			if lastIndex > additionalBPsLen {
				continue
			}
			if !rdgValid { // Earliest reading.
				// Set invalid reading for break points without reading.
				for i := lastIndex; i < additionalBPsLen; i++ {
					bp := additionalBPs[i]
					bpActual := bp[1]
					_, ok := bpRdgs[bpActual]
					if !ok {
						bpRdgs[bpActual] = make(map[int32]*Rdg)
					}
					_, ok = bpRdgs[bpActual][smID]
					if !ok {
						bpRdgs[bpActual][smID] = &Rdg{}
					}
				}
				continue
			}
			var lrValGE bool
			var lrVal float64
			lr, ok := laterRdgs[smID]
			if ok {
				lrVal = lr.Value
				if lrVal >= rdgVal {
					// Later reading value is greater or equal to current
					// reading value.
					lrValGE = true
				}
			}
			for i := lastIndex; i < additionalBPsLen; i++ {
				bp := additionalBPs[i]
				bpMin := bp[0]
				bpActual := bp[1]
				bpMax := bp[2]
				_, ok := bpRdgs[bpActual]
				if !ok {
					bpRdgs[bpActual] = make(map[int32]*Rdg)
				}
				prevBpRdg, prevBpRdgOk :=
					bpRdgs[bpActual][smID]
				if rdgTime.After(bpMax) {
					// After break point max.
					break
				} else if rdgTime.Compare(bpMin) >= 0 {
					// Between break point min and max.
					if !prevBpRdgOk || bpActual.Sub(rdgTime) <=
						bpActual.Sub(prevBpRdg.Time) {
						// Lower time difference or no previous reading.
						bpRdgs[bpActual][smID] = rdg
					}
					if bpActual.Compare(rdgTime) >= 0 {
						// No better reading possible.
						bpLastIndexes[smID] += 1
					}
				} else {
					// Before break point min.
					if lrValGE {
						rdgDayDiff := lr.Time.Sub(rdgTime).Hours() / 24
						newValPerDay := (lrVal - rdgVal) / rdgDayDiff
						newVal :=
							rdgVal + (newValPerDay * bpActual.Sub(rdgTime).Hours() / 24)
						newRdg := &Rdg{
							Value: newVal,
							Time:  bpActual,
							Valid: true,
						}
						bpRdgs[bpActual][smID] = newRdg
						bpLastIndexes[smID] += 1
					} else {
						// Set invalid reading for break point.
						bpRdgs[bpActual][smID] = &Rdg{}
						bpLastIndexes[smID] += 1
					}
				}
			}
			laterRdgs[smID] = rdg
		}
		// Merge all break points.
		calcBPs = append(calcBPs, additionalBPs...)
		sort.Sort(sort.Reverse(calcBPs))
	}

	smList, err := qtx.ListSms(ctx, mmID)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	sms := make(map[int32]spinusdb.ListSmsRow)
	for _, sm := range smList {
		sms[sm.ID] = sm
	}

	var smFinBalances []spinusdb.UpdateSmFinBalanceParams
	smBills := make(map[int32]*spinusdb.CreateSmBillParams)
	smBillPeriods := make(map[int]map[int32]*spinusdb.CreateSmBillPeriodParams)

	calcBPsLen = len(calcBPs)

	calcBpIndex := 0
	bp := calcBPs[calcBpIndex]
	laterBPRdgs := bpRdgs[bp[1]]
	calcBpIndex++

	smLen := len(bpRdgs[calcBPs[calcBpIndex][1]])

	// Prepare main meter billing period.
	var mmServicePriceValid bool
	var mmServicePricePerSm float64
	mmBillPeriodIndex = 0
	smBillPeriods[mmBillPeriodIndex] = make(
		map[int32]*spinusdb.CreateSmBillPeriodParams)
	mmBillPeriod := mmBillPeriods[mmBillPeriodIndex]
	mmBeginTime := mmBillPeriod.BeginDate.Time
	mmMinTime := mmBeginTime.AddDate(0, 0, -1) // Shifted one day back.
	mmEndTime := mmBillPeriod.EndDate.Time
	mmDays := mmEndTime.Sub(mmMinTime).Hours() / 24
	mmBeginVal := mmBillPeriod.BeginRdgVal
	mmEndVal := mmBillPeriod.EndRdgVal
	mmConsum := mmEndVal - mmBeginVal
	mmValPerDay := (mmEndVal - mmBeginVal) / mmDays
	mmConsumEnergyPricePerUnit := mmBillPeriod.ConsumEnergyPrice / mmConsum
	if mmBillPeriod.ServicePrice.Valid {
		mmServicePriceValid = true
		mmServicePricePerSm = mmBillPeriod.ServicePrice.Float64 / float64(smLen)
	}
	mmLaterBpVal := mmEndVal // Main meter later break point value.

	for ; calcBpIndex < calcBPsLen; calcBpIndex++ { // From latest to earliest.
		bp := calcBPs[calcBpIndex]
		bpActual := bp[1]
		var mmBpVal float64
		var mmBpConsum float64
		if bpActual.After(mmMinTime) {
			// Actual is between main meter billing period min and max.
			// Need to calculate main meter reading value.
			mmBpVal = mmBeginVal + (mmValPerDay * bpActual.Sub(mmMinTime).Hours() / 24)
		} else {
			mmBpVal = mmBeginVal
		}
		// Calculate main meter break point consumption.
		mmBpConsum = mmLaterBpVal - mmBpVal
		mmLaterBpVal = mmBpVal
		var invalidCount int
		var rdgsSum, laterRdgsSum float64
		rdgs := bpRdgs[bpActual]
		for smID, rdg := range rdgs {
			// Calculate invalid count and sums for current and later break point.
			// Sum must be calculated every time even for later break point
			// because later reading can be lower then current reading.
			laterRdg := laterBPRdgs[smID]
			if !rdg.Valid || !laterRdg.Valid {
				invalidCount++
				continue
			}
			rdgVal := rdg.Value
			laterRdgVal := laterRdg.Value
			if rdgVal > laterRdgVal {
				invalidCount++
			} else {
				rdgsSum += rdgVal
				laterRdgsSum += laterRdgVal
			}

		}
		var energyConsumValidAddendum float64
		var energyConsumInvalidAddendum float64
		if invalidCount == 0 {
			// All valid, split difference equally to all sub meters.
			energyConsumValidAddendum =
				(mmBpConsum - (laterRdgsSum - rdgsSum)) / float64(smLen)
		} else {
			// At least one invalid, split difference only to sub meters with invalid
			// reading.
			energyConsumInvalidAddendum =
				(mmBpConsum - (laterRdgsSum - rdgsSum)) / float64(invalidCount)
		}
		for smID, rdg := range rdgs {
			// Calculate energy consumption and energy consumption price.
			smBillPeriod, ok :=
				smBillPeriods[mmBillPeriodIndex][smID]
			if !ok {
				smBillPeriod = &spinusdb.CreateSmBillPeriodParams{}
				if mmServicePriceValid {
					smBillPeriod.ServicePrice = pgtype.Float8{
						Float64: mmServicePricePerSm, Valid: true}
				}
				smBillPeriods[mmBillPeriodIndex][smID] =
					smBillPeriod
			}
			var energyConsum float64
			laterRdg := laterBPRdgs[smID]
			if !rdg.Valid || !laterRdg.Valid {
				energyConsum = energyConsumInvalidAddendum
			} else {
				rdgVal := rdg.Value
				laterRdgVal := laterRdg.Value
				if rdgVal > laterRdgVal {
					energyConsum = energyConsumInvalidAddendum
				} else {
					energyConsum = laterRdgVal - rdgVal + energyConsumValidAddendum
				}
			}
			smBillPeriod.EnergyConsum += energyConsum
			smBillPeriod.ConsumEnergyPrice +=
				energyConsum * mmConsumEnergyPricePerUnit
		}
		if bpActual.Equal(mmMinTime) {
			// Earliest break point for current main meter billing period.
			smBillPrds := smBillPeriods[mmBillPeriodIndex]
			for smID, smBillPeriod := range smBillPrds {
				// Calculate all prices for sub meter billing periods and main
				// meter billing period.
				smBill, ok := smBills[smID]
				var smForm *SmBillForm
				if ok {
					smForm = smIDBillForms[smID]
				} else {
					sm := sms[smID]
					smBill = &spinusdb.CreateSmBillParams{FkSm: smID}
					smBills[smID] = smBill
					smForm = &SmBillForm{
						ID:      smID,
						Subid:   sm.Subid,
						MeterID: sm.MeterID,
						Email:   sm.Email,
					}
					smIDBillForms[smID] = smForm
					tmplData.SmBills = append(
						tmplData.SmBills, smForm)
				}

				energyConsum := smBillPeriod.EnergyConsum
				consumEnergyPrice := smBillPeriod.ConsumEnergyPrice
				var servicePrice float64
				if mmServicePriceValid {
					servicePrice = smBillPeriod.ServicePrice.Float64
					smBill.ServicePrice.Float64 += servicePrice
					smBill.ServicePrice.Valid = true
					smForm.ServicePrice.Float64 += servicePrice
					smForm.ServicePrice.Valid = true
				}
				advancePrice := consumEnergyPrice + servicePrice
				totalPrice := consumEnergyPrice + servicePrice + advancePrice
				smBillPeriod.AdvancePrice = advancePrice
				smBillPeriod.TotalPrice = totalPrice
				smBill.EnergyConsum += energyConsum
				smBill.ConsumEnergyPrice += consumEnergyPrice
				smBill.AdvancePrice += advancePrice
				smForm.EnergyConsum += energyConsum
				smForm.ConsumEnergyPrice += consumEnergyPrice
				smForm.AdvancePrice += advancePrice
				// smForm.TotalPrice += totalPrice
				mmBillPeriod.AdvancePrice += advancePrice
				mmBillPeriod.TotalPrice += advancePrice
			}
			// Update main meter billing.
			mmBillPrdAdvancePrice := mmBillPeriod.AdvancePrice
			mmBill.AdvancePrice += mmBillPrdAdvancePrice
			mmBillPeriodIndex++
			if mmBillPeriodIndex == billPeriodsLen {
				allSmBillsPaid := true
				for smID, smBill := range smBills {
					consumEnergyPrice := smBill.ConsumEnergyPrice
					smForm := smIDBillForms[smID]
					var servicePrice float64
					if smBill.ServicePrice.Valid {
						servicePrice = smBill.ServicePrice.Float64
					}
					advancePrice := smBill.AdvancePrice
					totalPrice := consumEnergyPrice + servicePrice +
						advancePrice
					sm := sms[smID]
					finBal := sm.FinBalance
					if totalPrice <= finBal {
						fromFinBal := -totalPrice
						toPay := 0.0
						mmBill.FromFinBalance += fromFinBal
						mmBill.ToPay += toPay
						smBill.FromFinBalance = fromFinBal
						smBill.ToPay = toPay
						smBill.Status = spinusdb.SmBillStatusPaid
						smForm.FromFinBalance = fromFinBal
						smForm.ToPay = toPay
						newFinBal := finBal - (consumEnergyPrice + servicePrice)
						smFinBalance :=
							spinusdb.UpdateSmFinBalanceParams{
								ID:         smID,
								FinBalance: newFinBal,
							}
						smFinBalances = append(
							smFinBalances,
							smFinBalance,
						)
					} else {
						fromFinBalance := -finBal
						toPay := totalPrice - finBal
						mmBill.FromFinBalance += fromFinBalance
						mmBill.ToPay += toPay
						smBill.FromFinBalance = fromFinBalance
						smBill.ToPay = toPay
						smBill.Status = spinusdb.SmBillStatusUnpaid
						smForm.FromFinBalance = fromFinBalance
						smForm.ToPay = toPay
						allSmBillsPaid = false
					}
				}
				if allSmBillsPaid {
					mmBill.Status = spinusdb.MmBillStatusCompleted
				} else {
					mmBill.Status = spinusdb.MmBillStatusInprogress
				}
				break
			}

			// Prepare main meter billing period.
			smBillPeriods[mmBillPeriodIndex] = make(
				map[int32]*spinusdb.CreateSmBillPeriodParams)
			mmBillPeriod = mmBillPeriods[mmBillPeriodIndex]
			mmBeginTime = mmBillPeriod.BeginDate.Time
			mmMinTime = mmBeginTime.AddDate(0, 0, -1)
			mmEndTime = mmBillPeriod.EndDate.Time
			mmDays = mmEndTime.Sub(mmMinTime).Hours() / 24
			mmBeginVal = mmBillPeriod.BeginRdgVal
			mmEndVal = mmBillPeriod.EndRdgVal
			mmConsum = mmEndVal - mmBeginVal
			mmValPerDay = (mmEndVal - mmBeginVal) / mmDays
			mmConsumEnergyPricePerUnit =
				mmBillPeriod.ConsumEnergyPrice / mmConsum
			mmServicePricePerSm =
				mmBillPeriod.ServicePrice.Float64 / float64(smLen)
			if mmBillPeriod.ServicePrice.Valid {
				mmServicePriceValid = true
				mmServicePricePerSm =
					mmBillPeriod.ServicePrice.Float64 / float64(smLen)
			}
			mmLaterBpVal = mmEndVal
		}
		laterBPRdgs = bpRdgs[bpActual]
	}

	// Calculate billing, do not create.
	if r.PostFormValue("calculate-bill") != "" {
		tmplData.Calculated = true
		sort.Sort(tmplData.SmBills)
		s.renderTemplate(w, r, tmplName, tmplData)
		return
	}
	createdMmBill, err := qtx.CreateMmBill(ctx, mmBill)
	if err != nil {
		slog.Error("error executing query", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}
	createdMmBillID := createdMmBill.ID

	createdSmBillIDs := make(map[int32]int32)

	for smID, smBill := range smBills {
		smBill.FkMmBill = createdMmBillID
		createdSmBill, err := qtx.CreateSmBill(ctx, *smBill)
		if err != nil {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		createdSmBillIDs[smID] = createdSmBill.ID
	}

	for i, mmBillPrd := range mmBillPeriods {
		mmBillPrd.FkMmBill = createdMmBillID
		createdMmBillPeriod, err := qtx.CreateMmBillPeriod(
			ctx, *mmBillPrd)
		if err != nil {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		createdMmBillPeriodID := createdMmBillPeriod.ID
		smBillPrds := smBillPeriods[i]
		for smID, smBillPrd := range smBillPrds {
			smBillPrd.FkSmBill = createdSmBillIDs[smID]
			smBillPrd.FkMmBillPeriod = createdMmBillPeriodID
			_, err := qtx.CreateSmBillPeriod(ctx, *smBillPrd)
			if err != nil {
				slog.Error("error executing query", "err", err)
				s.HandleInternalServerError(w, r, err)
				return
			}
		}
	}
	for _, smFinBalance := range smFinBalances {
		err := qtx.UpdateSmFinBalance(ctx, smFinBalance)
		if err != nil {
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("error committing transaction", "err", err)
		s.HandleInternalServerError(w, r, err)
		return
	}

	http.Redirect(
		w,
		r,
		fmt.Sprintf(
			"/main-meter/%d/billing/%d/overview",
			mm.ID,
			createdMmBillID,
		),
		http.StatusSeeOther,
	)
}
