package commands

import (
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

type signals struct {
	Current string `json:"current"` // current cart ID
	Name    string `json:"name"`    // name for current cart
	Msg      string `json:"msg"`      // unified input text (add item or chat)
	ChatMode string `json:"chatMode"` // "add" or "chat"
}

// we're just going to panic on error, for simplicity
func SignalsFromRequest(r *http.Request) *signals {
	var s signals
	if err := datastar.ReadSignals(r, &s); err != nil {
		panic("failed to read signals: " + err.Error())
	}
	return &s
}

// TrySignalsFromRequest returns nil instead of panicking on error.
func TrySignalsFromRequest(r *http.Request) *signals {
	var s signals
	if err := datastar.ReadSignals(r, &s); err != nil {
		return nil
	}
	return &s
}
