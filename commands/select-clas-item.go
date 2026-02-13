package commands

import (
	"net/http"
	"strconv"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
)

func NewSelectClasItem(
	repo *carts.SqliteRepository,
	bus *events.Bus,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromRequest(r)
		ID := r.URL.Query().Get("id")
		index, err := strconv.Atoi(r.URL.Query().Get("index"))
		if err != nil {
			log.Error("invalid index", "error", err)
			return
		}

		log.Info("setClasItem called", "id", ID, "index", index)

	}
}
