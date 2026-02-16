package commands

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/events"
)

func NewChatOpened(bus *events.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromRequest(r)
		cartID := chi.URLParam(r, "id")
		bus.Publish(events.ChatOpened{CartID: cartID, UserID: claims.UserID})
		w.WriteHeader(http.StatusNoContent)
	}
}
