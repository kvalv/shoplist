package commands

import (
	"context"
	"net/http"
	"net/url"

	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
	"github.com/kvalv/shoplist/recipe"
	"github.com/starfederation/datastar-go/datastar"
)

func NewAddItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		signals := SignalsFromRequest(r)
		claims := auth.ClaimsFromRequest(r)
		cartID := signals.Current

		if signals.Mode == "chat" {
			msg := carts.NewMessage(cartID, signals.Msg).WithUser(claims.UserID)
			log.Info("/add (chat)", "text", signals.Msg, "user", claims.UserID)
			if err := repo.AddMessage(msg); err != nil {
				log.Error("failed to add message", "error", err)
				return
			}
			bus.Publish(events.CartUpdated{CartID: cartID})
			bus.Publish(events.ChatAdded{CartID: cartID, MessageID: msg.ID})
			datastar.NewSSE(w, r).PatchSignals([]byte(`{"msg": ""}`))
			return
		}

		cart, err := repo.Cart(cartID)
		if err != nil {
			log.Error("failed to fetch cart", "error", err, "cartID", cartID)
			return
		}
		log.Info("/add invoked", "text", signals.Msg, "cartID", cart.ID)

		event := events.CartUpdated{
			CartID: cart.ID,
		}
		if got, _ := url.ParseRequestURI(signals.Msg); got != nil {
			log.Info("this is a recipe, trying to parse")
			parts, err := recipe.Parse(context.Background(), got)
			if err != nil {
				log.Error("failed to parse recipe", "error", err)
			}
			log.Info("parsed recipe", "parts", len(parts))
			for _, text := range parts {
				log.Info("adding item from recipe", "text", text)
				item := cart.Add(text, claims.UserID)
				event.ItemIDs = append(event.ItemIDs, item.ID)
			}
		} else {
			item := cart.Add(signals.Msg, claims.UserID)
			event.ItemIDs = append(event.ItemIDs, item.ID)
		}
		repo.Save(cart)
		bus.Publish(event)
		datastar.NewSSE(w, r).PatchSignals([]byte(`{"msg": ""}`))
	}
}
