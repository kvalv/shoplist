package commands

import (
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
)

func NewSetName(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signals := SignalsFromRequest(r)
		cartID := signals.Current
		cart, err := repo.Cart(cartID)
		if err != nil {
			panic("wtf do we do here then")
		}
		cart.Name = signals.Name
		log.Info("Cart renamed", "new", cart.Name)

		if err := repo.Save(cart); err != nil {
			panic("wtf do we do here then")
		}

		bus.Publish(events.CartUpdated{CartID: cartID})
	}
}
