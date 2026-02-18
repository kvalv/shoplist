package commands

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
	"github.com/kvalv/shoplist/stores"
	"github.com/starfederation/datastar-go/datastar"
)

func NewNewCart(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		claims := auth.ClaimsFromRequest(r)
		signals := SignalsFromRequest(r)

		store, err := strconv.Atoi(signals.NewCartStore)
		if err != nil {
			log.Error("failed to parse store", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		name := time.Now().Format("2 January")
		cart := carts.New().WithName(name).WithCreator(claims.UserID)
		cart.TargetStore = stores.Store(store)
		if err := repo.Save(cart); err != nil {
			log.Error("failed to save cart", "error", err)
			http.Error(w, "failed to create cart", http.StatusInternalServerError)
			return
		}
		msg := carts.NewMessage(cart.ID, "Cart created")
		if claims.UserID != "" {
			msg.WithUser(claims.UserID)
		} else {
			msg.WithSystem()
		}
		if err := repo.AddMessage(msg); err != nil {
			log.Error("failed to add cart-created message", "error", err)
		}
		bus.Publish(events.CartCreated{CartID: cart.ID, CreatedBy: &claims.UserID})
		datastar.NewSSE(w, r).Redirect("/" + cart.ID)
	}
}
