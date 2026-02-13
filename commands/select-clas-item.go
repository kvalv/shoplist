package commands

import (
	"net/http"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewSelectClasItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		signals := SignalsFromRequest(r)
		ID := r.URL.Query().Get("id")
		clasID := r.URL.Query().Get("clasId")

		log.Info("setClasItem called", "id", ID, "clasId", clasID)
		if err := repo.SelectClasOhlsonItem(ID, clasID); err != nil {
			log.Error("failed to select clas ohlson item", "error", err)
			return
		}
		bus.Publish(events.CartUpdated{CartID: signals.Current, ItemIDs: []string{ID}})
	}
}
