package commands

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
)

func NewSwitchCart(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromRequest(r)
		signals := SignalsFromRequest(r)

		// for the special value '_new', we're actually creating and switching
		// to the new cart
		if signals.Current == "_new" {
			log.Info("would create new ")

			name := time.Now().Format("2 January")
			cart := carts.New().WithName(name).WithCreator(claims.UserID)

			if err := repo.Save(cart); err != nil {
				log.Error("failed to save cart", "error", err)
				http.Error(w, "failed to create cart", http.StatusInternalServerError)
				return
			}
			log.Info("new cart created", "cartID", cart.ID, "name", name, "createdBy", claims.UserID)
			bus.Publish(events.CartCreated{CartID: cart.ID})
			return

		}

		// TODO: something something active cart for a specific user
		log.Info("switch-cart called", "new", signals.Current)

		bus.Publish(events.CartSwitched{CartID: signals.Current, UserID: claims.UserID})
	}
}
