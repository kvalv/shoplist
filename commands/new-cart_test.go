package commands

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
)

func TestNewCart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := carts.NewTestRepository(t).WithUsers("alice")
		defer repo.Close()

		bus := events.NewBus(slog.Default())
		sub := bus.Subscribe()
		defer sub.Close()

		handler := NewNewCart(&repo.SqliteRepository, bus)

		body := strings.NewReader(`{"newCartStore":"0"}`)
		r := httptest.NewRequest(http.MethodPost, "/new-cart", body)
		r = r.WithContext(auth.WithClaims(r.Context(), &auth.Claims{UserID: "alice"}))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		ev := <-sub.Ch
		created, ok := ev.(events.CartCreated)
		if !ok {
			t.Fatalf("expected CartCreated event, got %T", ev)
		}
		if created.CartID == "" {
			t.Fatal("expected non-empty CartID")
		}
		if created.CreatedBy == nil || *created.CreatedBy != "alice" {
			t.Fatalf("expected CreatedBy=alice, got %v", created.CreatedBy)
		}
	})
}
