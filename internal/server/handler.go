package server

import (
	"log/slog"
	"net/http"
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
