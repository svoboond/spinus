package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/svoboond/spinus/internal/conf"
	"github.com/svoboond/spinus/internal/db"
	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
	"github.com/svoboond/spinus/internal/tmpl"
	"github.com/svoboond/spinus/ui"
)

type Server struct {
	server         *http.Server
	templates      tmpl.Template
	postgresClient *pgxpool.Pool
	queries        *spinusdb.Queries
	redisClient    *redis.Client
	sessionManager *scs.SessionManager
}

func New(config *conf.Conf) (*Server, error) {
	slog.Debug("parsing templates...")
	templates, err := tmpl.NewTemplateRenderer(
		ui.EmbeddedContentHTML, "html/*.html", "html/**/*.html")
	if err != nil {
		return nil, fmt.Errorf("could not create templates: %w", err)
	}

	slog.Debug("connecting to postgres...")
	postgresUrl := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", config.Postgres.Host, config.Postgres.Port),
		User:   url.UserPassword(config.Postgres.Username, config.Postgres.Password),
		Path:   config.Postgres.Name,
	}
	postgresCtx := context.Background()
	postgresClient, err := pgxpool.New(postgresCtx, postgresUrl.String())
	if err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	if err := migrateDatabase(postgresClient); err != nil {
		return nil, fmt.Errorf("could not migrate database: %w", err)
	}

	dbQueries := spinusdb.New(postgresClient)

	slog.Debug("connecting to redis...")
	redisOpt, err := redis.ParseURL(config.Redis.Url)
	if err != nil {
		return nil, fmt.Errorf("could not connect to redis: %w", err)
	}
	redisClient := redis.NewClient(redisOpt)

	sessionManager := scs.New()
	sessionManager.Store = goredisstore.New(redisClient)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Service.Port),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}

	router := chi.NewRouter()
	server.Handler = router

	app := &Server{
		server:         server,
		templates:      templates,
		postgresClient: postgresClient,
		queries:        dbQueries,
		redisClient:    redisClient,
		sessionManager: sessionManager,
	}

	// middlewares
	router.Use(chi_middleware.Recoverer)
	router.Use(chi_middleware.RealIP)
	router.Use(chi_middleware.Logger)
	router.Use(chi_middleware.RequestID)
	router.Use(sessionManager.LoadAndSave)
	router.Use(chi_middleware.Compress(5, "text/*", "application/*"))

	// handlers
	router.Handle("/static/*", WithCacheControl(
		http.FileServer(http.FS(ui.EmbeddedContentStatic)),
		31536000, // 1 year cache. We change file names if we update static files.
	))

	router.Get("/hello", app.HandleHelloGet)

	router.Get("/main-meter-list", app.HandleGetMainMeterList)
	router.Get("/main-meter-create", app.HandleGetMainMeterCreate)
	router.Post("/main-meter-create", app.HandlePostMainMeterCreate)

	return app, nil
}

func (s *Server) ListenAndServe() error { return s.server.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error {
	defer s.postgresClient.Close()
	defer s.redisClient.Close()
	return s.server.Shutdown(ctx)
}
func (s *Server) Close() error { return s.server.Close() }

func migrateDatabase(pool *pgxpool.Pool) error {
	slog.Debug("migrating database...")
	goose.SetBaseFS(db.EmbeddedContentMigration)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("could not set goose dialect: %w", err)
	}
	handle := stdlib.OpenDBFromPool(pool)
	if err := goose.Up(handle, "migration"); err != nil {
		return fmt.Errorf("could not execute database migration: %w", err)
	}
	if err := handle.Close(); err != nil {
		return fmt.Errorf("could not close database: %w", err)
	}
	return nil
}
