package homeassistant

import (
	"testing"

	"github.com/kvalv/shoplist/env"
)

func TestNotify(t *testing.T) {
	t.Skip("manual test — sends real push notification")

	cfg := env.Load()
	if cfg.HOMEASSISTANT_BASE_URL == "" || cfg.HOMEASSISTANT_TOKEN == "" {
		t.Skip("HOMEASSISTANT_BASE_URL / HOMEASSISTANT_TOKEN not set")
	}

	c := NewClient(cfg.HOMEASSISTANT_BASE_URL, cfg.HOMEASSISTANT_TOKEN)
	if err := c.Notify("Shoplist", "hi from claude"); err != nil {
		t.Fatal(err)
	}
}
