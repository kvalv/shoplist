package commands

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/stores"
	"github.com/starfederation/datastar-go/datastar"
)

func NewSetStore(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		var signals struct {
			Store string `json:"store"`
		}
		if err := datastar.ReadSignals(r, &signals); err != nil {
			log.Error("failed to read signals", "error", err)
			return
		}

		var err error
		if cart.TargetStore, err = parseStore(signals.Store); err != nil {
			log.Error("failed to parse", "error", err)
			return
		}
		repo.Save(cart)

		log.Info("/store called", "store", cart.TargetStore)
		bus.Publish(events.CartUpdated{CartID: cart.ID})
	}
}

func parseStore(s string) (stores.Store, error) {
	got, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return stores.Store(got), nil
}
