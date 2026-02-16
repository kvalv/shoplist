package commands

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewDiscardItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		itemID := chi.URLParam(r, "id")
		reason := chi.URLParam(r, "reason")
		userID := auth.ClaimsFromRequest(r).UserID

		cartID, err := repo.DiscardItem(itemID, userID)
		if err != nil {
			log.Error("failed to discard item", "error", err, "itemID", itemID)
			return
		}

		bus.Publish(events.CartUpdated{CartID: cartID})
		bus.Publish(events.ItemDiscarded{CartID: cartID, ItemID: itemID, UserID: userID, Reason: reason})
	}
}
