package commands

import (
	"net/http"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewDeleteItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		signals := SignalsFromRequest(r)
		ID := r.URL.Query().Get("id")

		log.Info("delete called", "id", ID)
		if err := repo.DeleteItem(ID); err != nil {
			log.Error("failed to delete item", "error", err)
			return
		}
		bus.Publish(events.CartUpdated{CartID: signals.Current})
	}
}
