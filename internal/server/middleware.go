package server

import (
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

func WithCacheControl(h http.Handler, maxAge int) http.Handler {
	cacheHeaderVal := fmt.Sprintf("public, max-age=%d", maxAge)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheHeaderVal)
		h.ServeHTTP(w, r)
	})
}

const userIDKey = "userID"
const emptyUserIDVal int32 = 0

func UserID(ctx context.Context) (int32, bool) {
	id, ok := ctx.Value(userIDKey).(int32)
	return id, ok
}

func (s *Server) WithUserID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := s.sessionManager.GetInt32(ctx, userIDKey)
		ctx = context.WithValue(ctx, userIDKey, userID)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) WithRequiredLogin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserID(r.Context())
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		if userID == emptyUserIDVal {
			query := r.URL.Query()
			query.Add("next", r.URL.Path)
			redirectUrl := url.URL{Path: "/login", RawQuery: query.Encode()}
			http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
			return
		}
		h.ServeHTTP(w, r)
	})
}

const mmKey = "mainMeter"

func GetMm(ctx context.Context) (spinusdb.GetMmRow, bool) {
	mm, ok := ctx.Value(mmKey).(spinusdb.GetMmRow)
	return mm, ok
}

func (s *Server) WithMm(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "mmID"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		mmID := int32(id)
		ctx := r.Context()
		userID, ok := UserID(ctx)
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		mm, err := s.queries.GetMm(ctx, mmID)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		if userID != mm.FkUser {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, mmKey, mm)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

const smKey = "subMeter"

func GetSm(ctx context.Context) (spinusdb.GetSmRow, bool) {
	sm, ok := ctx.Value(smKey).(spinusdb.GetSmRow)
	return sm, ok
}

func (s *Server) WithSm(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "mmID"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		mmID := int32(id)
		id, err = strconv.ParseInt(chi.URLParam(r, "subid"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		subid := int32(id)
		ctx := r.Context()
		userID, ok := UserID(ctx)
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		sm, err := s.queries.GetSm(
			ctx,
			spinusdb.GetSmParams{FkMm: mmID, Subid: subid},
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		if userID != sm.SubUserID || userID != sm.MainUserID {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, smKey, sm)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
