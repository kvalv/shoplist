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
		signals := SignalsFromRequest(r)
		ID := r.URL.Query().Get("id")
		userID := auth.ClaimsFromRequest(r).UserID

		cart, _ := repo.Cart(signals.Current)
		cart.Get(ID).Toggle(userID)
		repo.Save(cart)

		bus.Publish(events.CartUpdated{CartID: cart.ID})

		log.Info("tick called", "itemID", ID)
	}
}
