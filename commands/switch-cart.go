package commands

import (
	"log/slog"
	"net/http"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/starfederation/datastar-go/datastar"
)

type signals struct {
	Current string `json:"current"` // current cart ID
}

// we're just going to panic on error, for simplicity
func SignalsFromRequest(r *http.Request) *signals {
	var s signals
	if err := datastar.ReadSignals(r, s); err != nil {
		panic("failed to read signals: " + err.Error())
	}
	return &s
}

func NewSwitchCart(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signals := SignalsFromRequest(r)

		// TODO: something something active cart for a specific user
		log.Info("switch-cart called", "current", signals.Current)

		bus.Publish(events.CartSwitched{CartID: signals.Current})
	}
}
