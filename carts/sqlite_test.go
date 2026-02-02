package carts

import (
	"testing"

	"github.com/kvalv/shoplist/stores"
)

func TestSqliteBasic(t *testing.T) {
	repo := NewMock()

	// Create a cart
	cart := New()
	if err := repo.Save(New()); err != nil {
		t.Fatalf("Failed to save cart: %s", err)
	}
	t.Logf("Created cart: %+v", cart)

	// Fetch it back
	got, err := repo.Cart(cart.ID)
	if err != nil {
		t.Fatalf("Cart() error: %v", err)
	}
	t.Logf("Fetched cart: %+v", got)

	// List latest
	latest, err := repo.Latest()
	if err != nil {
		t.Fatalf("Latest() error: %v", err)
	}
	t.Logf("Latest cart: %+v", latest)
}

func TestSetupMockData(t *testing.T) {
	repo := NewMock()
	SetupMockData(repo)

	latest, err := repo.Latest()
	if err != nil {
		t.Fatalf("Latest() error: %v", err)
	}

	if latest.TargetStore != stores.Kiwi {
		t.Errorf("TargetStore = %v, want %v", latest.TargetStore, stores.Kiwi)
	}

	t.Logf("Cart: %+v", latest)
	for _, item := range latest.Items {
		t.Logf("  Item: %+v", item)
	}
}
