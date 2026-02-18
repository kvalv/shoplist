package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

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

// CloudflareAccessAuth reads the Cf-Access-Jwt-Assertion header set by
// Cloudflare Access and extracts user claims from the JWT payload.
// If the header is missing, falls back to the provided fallback middleware.
// The JWT signature is not verified here — we trust the tunnel.
func CloudflareAccessAuth(fallback *Claims, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Cf-Access-Jwt-Assertion")
			if token == "" {
				log.Info("no CF Access header, using fallback auth", "name", fallback.Name)
				next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), fallback)))
				return
			}

			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				http.Error(w, "malformed JWT", http.StatusUnauthorized)
				return
			}

			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				http.Error(w, "invalid JWT payload", http.StatusUnauthorized)
				return
			}

			var cfClaims struct {
				Sub     string `json:"sub"`
				Email   string `json:"email"`
				Country string `json:"country"`
			}
			if err := json.Unmarshal(payload, &cfClaims); err != nil {
				http.Error(w, "invalid JWT claims", http.StatusUnauthorized)
				return
			}

			claims := &Claims{
				UserID: cfClaims.Sub,
				Email:  cfClaims.Email,
				Name:   cfClaims.Email, // default to email
			}

			// Try to enrich with name/picture from CF get-identity endpoint
			if identity, ok := getIdentity(r, log); ok {
				if identity.Name != "" {
					claims.Name = identity.Name
				}
				if identity.picture() != "" {
					claims.Picture = identity.picture()
				}
			}

			log.Info("CF Access auth", "email", claims.Email, "name", claims.Name, "userID", claims.UserID, "country", cfClaims.Country)
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

var (
	identityCache   = map[string]*cfIdentity{}
	identityCacheMu sync.RWMutex
)

type cfIdentity struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	OIDCFields struct {
		Picture string `json:"picture"`
		Name    string `json:"name"`
	} `json:"oidc_fields"`
}

func (c *cfIdentity) picture() string { return c.OIDCFields.Picture }

// getIdentity calls the CF Access get-identity endpoint using the user's
// CF_Authorization cookie. Results are cached by email.
func getIdentity(r *http.Request, log *slog.Logger) (*cfIdentity, bool) {
	cookie, err := r.Cookie("CF_Authorization")
	if err != nil {
		return nil, false
	}

	// Check cache
	identityCacheMu.RLock()
	if cached, ok := identityCache[cookie.Value[:16]]; ok {
		identityCacheMu.RUnlock()
		return cached, true
	}
	identityCacheMu.RUnlock()

	// Call CF get-identity endpoint
	url := "https://" + r.Host + "/cdn-cgi/access/get-identity"
	req, _ := http.NewRequest("GET", url, nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warn("get-identity request failed", "error", err)
		return nil, false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Info("get-identity response", "status", resp.StatusCode, "body", string(body))

	var identity cfIdentity
	if err := json.Unmarshal(body, &identity); err != nil {
		log.Warn("get-identity decode failed", "error", err)
		return nil, false
	}

	// Cache it
	identityCacheMu.Lock()
	identityCache[cookie.Value[:16]] = &identity
	identityCacheMu.Unlock()

	return &identity, true
}

func NewMockAuth(claims *Claims) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
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
			if n > 0 {
				// Update name/picture in case they changed
				db.Exec("UPDATE users SET name = ?, picture = ? WHERE user_id = ?",
					claims.Name, claims.Picture, claims.UserID)
				next.ServeHTTP(w, r)
				return
			}

			if _, err := db.Exec("INSERT INTO users(user_id, name, email, picture) VALUES (?, ?, ?, ?)",
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
			time.Sleep(time.Millisecond * 100)
			next.ServeHTTP(w, r)
		})
	}
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ctxKey, claims)
}
