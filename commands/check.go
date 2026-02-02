package commands

import (
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/events"
)

func NewCheckItem(
	repo *cart.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		cart, _ := repo.Latest()
		cart.Get(ID).Toggle()
		repo.Save(cart)

		bus.Publish(events.CartUpdated{CartID: cart.ID})

		log.Info("tick called")
	}
}
