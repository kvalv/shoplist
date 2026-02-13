package logger

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/auth"
)

var ctxKey struct{}

// Middleware sets the base logger in the request context.
func Middleware(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxKey, base)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromRequest returns a logger with URI and userID attributes.
func FromRequest(r *http.Request) *slog.Logger {
	base, ok := r.Context().Value(ctxKey).(*slog.Logger)
	if !ok {
		base = slog.Default()
	}
	claims := auth.ClaimsFromRequest(r)
	return base.With("uri", r.URL.Path, "userID", claims.UserID)
}
