package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/svoboond/spinus/internal/conf"
	"github.com/svoboond/spinus/internal/db"
	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
	"github.com/svoboond/spinus/internal/server"
	"github.com/svoboond/spinus/internal/tmpl"
	"github.com/svoboond/spinus/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("during run", "err", err)
		os.Exit(1)
	}
}

func setupLogger(level, handler string) {
	handlerOpts := &slog.HandlerOptions{
		AddSource: true, Level: slog.LevelInfo}
	switch strings.ToLower(level) {
	case "debug":
		handlerOpts.Level = slog.LevelDebug
	case "info":
		handlerOpts.Level = slog.LevelInfo
	case "warn":
		handlerOpts.Level = slog.LevelWarn
	case "error":
		handlerOpts.Level = slog.LevelError
	}

	var slogHandler slog.Handler
	switch strings.ToLower(handler) {
	case "text":
		slogHandler = slog.NewTextHandler(os.Stderr, handlerOpts)
	case "json":
		slogHandler = slog.NewJSONHandler(os.Stderr, handlerOpts)
	default:
		slogHandler = slog.NewTextHandler(os.Stderr, handlerOpts)
	}

	slog.SetDefault(slog.New(slogHandler))
}

func migrateDb(pool *pgxpool.Pool) error {
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

func run() error {
	slog.Debug("configuring...")
	localConfPath := flag.String("config", "", "configuration file path")
	flag.Parse()
	config, err := conf.New(*localConfPath)
	if err != nil {
		return fmt.Errorf("could not create config: %w", err)
	}
	serverListenPort := config.Service.Port
	logLevel := config.Log.Level
	logHandler := config.Log.Handler

	setupLogger(logLevel, logHandler)

	slog.Debug("parsing templates...")
	templates, err := tmpl.NewTemplateRenderer(
		ui.EmbeddedContentHTML, "html/*.html", "html/**/*.html")
	if err != nil {
		return fmt.Errorf("could not create templates: %w", err)
	}

	slog.Debug("connecting to database...")
	ctx := context.Background()
	dbUrl := url.URL{
		Scheme: config.Database.Scheme,
		Host:   fmt.Sprintf("%s:%d", config.Database.Host, config.Database.Port),
		User:   url.UserPassword(config.Database.Username, config.Database.Password),
		Path:   config.Database.Name,
	}
	dbPool, err := pgxpool.New(ctx, dbUrl.String())
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer dbPool.Close()

	if err := migrateDb(dbPool); err != nil {
		return fmt.Errorf("could not migrate database: %w", err)
	}

	dbQueries := spinusdb.New(dbPool)

	slog.Debug("setting up server...")
	serverListenAddr := fmt.Sprintf(":%d", serverListenPort)
	appServer, err := server.New(
		serverListenAddr, templates, ui.EmbeddedContentStatic, dbQueries)
	if err != nil {
		return fmt.Errorf("could not create server: %w", err)
	}

	// graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("server startup", "status", "server starting", "addr", serverListenAddr)
		serverErrors <- appServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		slog.Info("server shutdown", "status", "shutdown started", "signal", sig.String())
		defer slog.Info(
			"server shutdown", "status", "shutdown complete", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := appServer.Shutdown(ctx); err != nil {
			appServer.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
