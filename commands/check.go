package commands

import (
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
)

func NewCheckItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		userID := auth.ClaimsFromRequest(r).UserID

		cart, _ := repo.Latest()
		cart.Get(ID).Toggle(userID)
		repo.Save(cart)

		bus.Publish(events.CartUpdated{CartID: cart.ID})

		log.Info("tick called")
	}
}
