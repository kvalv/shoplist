package commands

import (
	"net/http"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewCheckItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		itemID := r.URL.Query().Get("id")
		userID := auth.ClaimsFromRequest(r).UserID

		cartID, err := repo.ToggleItem(itemID, userID)
		if err != nil {
			log.Error("failed to toggle item", "error", err, "itemID", itemID)
			return
		}

		bus.Publish(events.CartUpdated{CartID: cartID})
		log.Info("tick called", "itemID", itemID)
	}
}
