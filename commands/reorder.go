package commands

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewReorderItems(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		cartID := chi.URLParam(r, "id")

		if err := repo.ReorderItems(cartID); err != nil {
			log.Error("failed to reorder items", "error", err, "cartID", cartID)
			return
		}

		bus.Publish(events.CartUpdated{CartID: cartID})
		log.Info("reorder called", "cartID", cartID)
	}
}
