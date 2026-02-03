package commands

import (
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

type signals struct {
	Current string `json:"current"` // current cart ID
	Name    string `json:"name"`    // name for current cart
}

// we're just going to panic on error, for simplicity
func SignalsFromRequest(r *http.Request) *signals {
	var s signals
	if err := datastar.ReadSignals(r, &s); err != nil {
		panic("failed to read signals: " + err.Error())
	}
	return &s
}
