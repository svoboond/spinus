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
const emptyUserIDValue int32 = 0

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
		if ok == false {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		if userID == emptyUserIDValue {
			query := r.URL.Query()
			query.Add("next", r.URL.Path)
			redirectUrl := url.URL{Path: "/login", RawQuery: query.Encode()}
			http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
			return
		}
		h.ServeHTTP(w, r)
	})
}

const mainMeterKey = "mainMeter"

func GetMainMeter(ctx context.Context) (spinusdb.GetMainMeterRow, bool) {
	mainMeter, ok := ctx.Value(mainMeterKey).(spinusdb.GetMainMeterRow)
	return mainMeter, ok
}

func (s *Server) WithMainMeter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "mainMeterID"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		mainMeterID := int32(id)
		ctx := r.Context()
		userID, ok := UserID(ctx)
		if ok == false {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		mainMeter, err := s.queries.GetMainMeter(ctx, mainMeterID)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r, err)
			return
		}
		if userID != mainMeter.FkUser {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, mainMeterKey, mainMeter)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

const subMeterKey = "subMeter"

func GetSubMeter(ctx context.Context) (spinusdb.GetSubMeterRow, bool) {
	subMeter, ok := ctx.Value(subMeterKey).(spinusdb.GetSubMeterRow)
	return subMeter, ok
}

func (s *Server) WithSubMeter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "mainMeterID"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		mainMeterID := int32(id)
		id, err = strconv.ParseInt(chi.URLParam(r, "subMeterID"), 10, 32)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		subMeterID := int32(id)
		ctx := r.Context()
		userID, ok := UserID(ctx)
		if ok == false {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r, errors.New("error getting user ID"))
			return
		}
		subMeter, err := s.queries.GetSubMeter(
			ctx,
			spinusdb.GetSubMeterParams{FkMainMeter: mainMeterID, Subid: subMeterID},
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
		if userID != subMeter.SubUserID || userID != subMeter.MainUserID {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, subMeterKey, subMeter)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
