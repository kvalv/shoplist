package commands

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/recipe"
	"github.com/starfederation/datastar-go/datastar"
)

func NewAddItem(
	repo *cart.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signals struct {
			Text string `json:"text"`
		}
		if err := datastar.ReadSignals(r, &signals); err != nil {
			log.Error("failed to read signals", "error", err)
		}
		cart, _ := repo.Latest()
		log.Info("/add invoked", "text", signals.Text, "cartID", cart.ID)

		event := events.CartUpdated{
			CartID: cart.ID,
		}
		if got, _ := url.ParseRequestURI(signals.Text); got != nil {
			log.Info("this is a recipe, trying to parse")
			parts, err := recipe.Parse(context.Background(), got)
			if err != nil {
				log.Error("failed to parse recipe", "error", err)
			}
			log.Info("parsed recipe", "parts", len(parts))
			for _, text := range parts {
				log.Info("adding item from recipe", "text", text)
				item := cart.Add(text)
				event.ItemIDs = append(event.ItemIDs, item.ID)
			}
		} else {
			item := cart.Add(signals.Text)
			event.ItemIDs = append(event.ItemIDs, item.ID)
		}
		repo.Save(cart)
		bus.Publish(event)
		datastar.NewSSE(w, r).PatchSignals([]byte(`{"text": ""}`))
	}
}
