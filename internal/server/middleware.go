package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
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

type userIDCtx string

const userIDKey userIDCtx = "userID"

var emptyUserIDVal uuid.UUID

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func (s *Server) WithUserID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sUserID := s.sessionManager.GetString(ctx, string(userIDKey))
		var v uuid.UUID
		userID, err := uuid.Parse(sUserID)
		if err != nil {
			userID = v
		}
		ctx = context.WithValue(ctx, userIDKey, userID)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) WithRequiredLogin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r)
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

type mmCtx string

const mmKey mmCtx = "mainMeter"

func GetMm(ctx context.Context) (spinusdb.GetMmRow, bool) {
	mm, ok := ctx.Value(mmKey).(spinusdb.GetMmRow)
	return mm, ok
}

func (s *Server) WithMm(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mmID, err := GetUUIDUrlParam(r)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		ctx := r.Context()
		userID, ok := GetUserID(ctx)
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r)
			return
		}
		mm, err := s.queries.GetMm(ctx, mmID)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r)
			return
		}
		if userID != mm.FkUser {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, mmKey, mm)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

type smCtx string

const smKey smCtx = "subMeter"

func GetSm(ctx context.Context) (spinusdb.GetSmRow, bool) {
	sm, ok := ctx.Value(smKey).(spinusdb.GetSmRow)
	return sm, ok
}

func (s *Server) WithSm(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		smID, err := GetUUIDUrlParam(r)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		ctx := r.Context()
		userID, ok := GetUserID(ctx)
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r)
			return
		}
		sm, err := s.queries.GetSm(ctx, smID)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r)
			return
		}
		if userID != sm.SubUserID || userID != sm.MainUserID {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, smKey, sm)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

type mmBillCtx string

const mmBillKey mmBillCtx = "mainMeterBilling"

func GetMmBill(ctx context.Context) (spinusdb.GetMmBillRow, bool) {
	mmBill, ok := ctx.Value(mmBillKey).(spinusdb.GetMmBillRow)
	return mmBill, ok
}

func (s *Server) WithMmBill(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mmBillID, err := GetUUIDUrlParam(r)
		if err != nil {
			s.HandleNotFound(w, r)
			return
		}
		ctx := r.Context()
		userID, ok := GetUserID(ctx)
		if !ok {
			slog.Error("error getting user ID", "userID", userID)
			s.HandleInternalServerError(w, r)
			return
		}
		mmBill, err := s.queries.GetMmBill(ctx, mmBillID)
		if err != nil {
			if err == pgx.ErrNoRows {
				s.HandleNotFound(w, r)
				return
			}
			slog.Error("error executing query", "err", err)
			s.HandleInternalServerError(w, r)
			return
		}
		if userID != mmBill.FkUser {
			s.HandleForbidden(w, r)
			return
		}
		ctx = context.WithValue(ctx, mmBillKey, mmBill)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
