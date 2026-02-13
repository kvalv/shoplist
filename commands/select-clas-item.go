package commands

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
)

func NewSelectClasItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		index, err := strconv.Atoi(r.URL.Query().Get("index"))
		if err != nil {
			log.Error("invalid index", "error", err)
			return
		}

		log.Info("setClasItem called", "id", ID, "index", index)

	}
}
