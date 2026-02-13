package logger

import (
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/auth"
)

// FromRequest returns a logger with URI and userID attributes.
func FromRequest(r *http.Request) *slog.Logger {
	claims := auth.ClaimsFromRequest(r)
	return slog.Default().With("uri", r.URL.Path, "userID", claims.UserID)
}
