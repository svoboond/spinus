package server

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	db "github.com/svoboond/spinus/internal/db/sqlc"
	"github.com/svoboond/spinus/internal/tmpl"
)

type Server struct {
	server    *http.Server
	templates tmpl.Template
	queries   *db.Queries
}

func New(
	listenAddr string, templates tmpl.Template, staticFS embed.FS, queries *db.Queries,
) (*Server, error) {
	server := &http.Server{
		Addr:              listenAddr,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}

	router := chi.NewRouter()
	server.Handler = router

	app := &Server{
		server:    server,
		templates: templates,
		queries:   queries,
	}

	// middlewares
	router.Use(chi_middleware.Recoverer)
	router.Use(chi_middleware.RealIP)
	router.Use(chi_middleware.Logger)
	router.Use(chi_middleware.RequestID)
	router.Use(chi_middleware.Compress(5, "text/*", "application/*"))

	// handlers
	router.Handle("/static/*", WithCacheControl(
		http.FileServer(http.FS(staticFS)),
		31536000, // 1 year cache. We change file names if we update static files.
	))

	router.Get("/hello", app.HandleHelloGet)

	router.Get("/main-meter-list", app.HandleGetMainMeterList)

	return app, nil
}

func (s *Server) ListenAndServe() error              { return s.server.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
func (s *Server) Close() error                       { return s.server.Close() }
