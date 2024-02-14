package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

func (s *Server) HandleHelloGet(w http.ResponseWriter, _ *http.Request) {
	const tmplName = "hello"

	if err := s.templates.Render(w, tmplName, nil); err != nil {
		slog.Error("HandleHelloGet template render", "template", tmplName, "err", err)
		return
	}
}

func (s *Server) HandleGetMainMeterList(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterList"

	mainMeters, err := s.queries.ListMainMeters(r.Context())
	if err != nil {
		slog.Error("HandleGetMainMeterList select", "err", err)
		return
	}

	if err := s.templates.Render(w, tmplName, mainMeters); err != nil {
		slog.Error(
			"HandleGetMainMeterList template render", "template", tmplName, "err", err)
		return
	}
}

func (s *Server) HandleGetMainMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterCreate"

	if err := s.templates.Render(w, tmplName, nil); err != nil {
		slog.Error(
			"HandleGetMainMeterCreate template render",
			"template", tmplName, "err", err,
		)
		return
	}
}

type MainMeterForm struct {
	No      string
	Address string
	Errors  map[string]string
}

func (s *Server) HandlePostMainMeterCreate(w http.ResponseWriter, r *http.Request) {
	const tmplName = "mainMeterCreate"

	form := MainMeterForm{}
	form.Errors = make(map[string]string)
	if err := r.ParseForm(); err != nil {
		slog.Info("HandlePostMainMeterCreate parse form", "err", err)
		form.Errors["General"] = "Bad request"
		if err := s.templates.Render(w, tmplName, form); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"HandleGetMainMeterCreate template render",
				"template", tmplName, "err", err,
			)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	number := r.PostFormValue("number")
	form.No = number
	no, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		form.Errors["No"] = "Enter number"
	}
	if no < 1000 {
		form.Errors["No"] = "Enter number no less than 1000"
	}
	address := r.PostFormValue("address")
	form.Address = address
	if address == "" {
		form.Errors["Address"] = "Enter address"
	} else if len(address) < 8 {
		form.Errors["Address"] = "Enter address with at least 8 characters"
	}
	if len(form.Errors) > 0 {
		if err := s.templates.Render(w, tmplName, form); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"HandleGetMainMeterCreate template render",
				"template", tmplName, "err", err,
			)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mainMeter, err := s.queries.CreateMainMeter(
		r.Context(), spinusdb.CreateMainMeterParams{No: no, Address: address})
	if err != nil {
		slog.Error("HandlePostMainMeterCreate query", "err", err)
		return
	}
	fmt.Println("hello", mainMeter.ID)

	http.Redirect(w, r, "/main-meter-list", http.StatusSeeOther)
}
