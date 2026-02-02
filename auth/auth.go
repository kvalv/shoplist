package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/events"
)

type Claims struct {
	UserID  string
	Name    string
	Email   string
	Picture string
}

var ctxKey struct{}

func ClaimsFromRequest(r *http.Request) *Claims {
	claims, ok := r.Context().Value(ctxKey).(*Claims)
	if !ok {
		panic("ClaimsFromRequest: value is not *Claims")
	}
	return claims
}

func CloudflareAccessAuth() func(http.Handler) http.Handler {
	// Idea: we'll use cloudflare access as our IDP, and reads
	// JWT. Docs: https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/
	// We'll map that to our own Claims that we're using.
	panic("Not implemented")
}

func NewMockAuth(claims *Claims) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
		})
	}
}

// A middleware that registers new users to the users table.
func RegisterUsers(
	db *sql.DB,
	log *slog.Logger,
	bus *events.Bus,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromRequest(r)

			rows, err := db.Query("select * from users where user_id = ?", claims.UserID)
			if err != nil {
				w.WriteHeader(500)
				log.Error(fmt.Sprintf("failed to execute sql query: %v", err))
				return
			}

			var n int
			for rows.Next() {
				n++
			}

			if _, err := db.Exec("insert into users(user_id, name, email, picture) values (?, ?, ?, ?)",
				claims.UserID,
				claims.Name,
				claims.Email,
				claims.Picture,
			); err != nil {
				log.Error(fmt.Sprintf("failed to insert user: %v", err))
				w.WriteHeader(500)
				return
			}
			bus.Publish(events.UserRegistered{UserID: claims.UserID})
			next.ServeHTTP(w, r)
		})
	}
}

func withClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ctxKey, claims)
}
