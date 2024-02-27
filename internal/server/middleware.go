package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func WithCacheControl(h http.Handler, maxAge int) http.Handler {
	cacheHeaderVal := fmt.Sprintf("public, max-age=%d", maxAge)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheHeaderVal)
		h.ServeHTTP(w, r)
	})
}

const userIdKey = "userId"
const emptyUserIdValue int32 = 0

func (s *Server) WithUserId(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId := s.sessionManager.GetInt32(ctx, userIdKey)
		ctx = context.WithValue(ctx, userIdKey, userId)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) WithRequiredLogin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(userIdKey) == emptyUserIdValue {
			query := r.URL.Query()
			query.Add("next", r.URL.Path)
			redirectUrl := url.URL{Path: "/login", RawQuery: query.Encode()}
			http.Redirect(w, r, redirectUrl.String(), http.StatusSeeOther)
			return
		}
		h.ServeHTTP(w, r)
	})
}
