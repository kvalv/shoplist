package commands

import (
	"net/http"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
	"github.com/starfederation/datastar-go/datastar"
)

func NewAddMessage(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		signals := SignalsFromRequest(r)
		claims := auth.ClaimsFromRequest(r)

		cart, _ := repo.Latest()
		msg := carts.NewMessage(cart.ID, signals.ChatMsg).WithUser(claims.UserID)
		log.Info("/add-message", "text", signals.ChatMsg, "user", claims.UserID)

		if err := repo.AddMessage(msg); err != nil {
			log.Error("failed to add message", "error", err)
			return
		}
		bus.Publish(events.CartUpdated{CartID: cart.ID})
		bus.Publish(events.ChatAdded{CartID: cart.ID, MessageID: msg.ID})
		datastar.NewSSE(w, r).PatchSignals([]byte(`{"chatMsg": ""}`))
	}
}
