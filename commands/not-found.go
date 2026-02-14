package commands

import (
	"net/http"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewNotFound(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		signals := SignalsFromRequest(r)
		claims := auth.ClaimsFromRequest(r)
		itemID := r.URL.Query().Get("id")

		cart, err := repo.Cart(signals.Current)
		if err != nil {
			log.Error("failed to get cart", "error", err)
			return
		}

		item := cart.Get(itemID)
		if item == nil {
			log.Error("item not found", "itemID", itemID)
			return
		}

		name := item.Text
		if selected := item.Clas.Selected(); selected != nil {
			name = selected.Name
		}

		msg := carts.NewMessage(signals.Current, "Fant ikke").WithUser(claims.UserID).WithItem(itemID)
		if err := repo.AddMessage(msg); err != nil {
			log.Error("failed to add message", "error", err)
			return
		}
		log.Info("/not-found", "item", name, "user", claims.UserID)
		bus.Publish(events.CartUpdated{CartID: signals.Current})
	}
}
