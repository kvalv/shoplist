package carts

import (
	"database/sql"
	"testing"

	"github.com/kvalv/shoplist/stores"
)

func TestSqliteBasic(t *testing.T) {
	repo, _ := NewMock()

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

func TestCollaborator(t *testing.T) {
	repo, _ := NewMock()

	t.Run("no collaborator", func(t *testing.T) {
		cart := New()
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "foo", false)
	})

	t.Run("with collaborator due to trigger", func(t *testing.T) {
		cart := New().WithCreator("user")
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "user", true)
	})

	t.Run("add collaborator", func(t *testing.T) {
		cart := New()
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "newuser", false)

		if err := repo.AddCollaborators(cart.ID, "newuser"); err != nil {
			t.Fatalf("AddCollaborator() error: %v", err)
		}
		expectCollaborator(t, repo, cart.ID, "newuser", true)
	})
}

func TestSetupMockData(t *testing.T) {
	repo, _ := NewMock()
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

func expectCollaborator(t *testing.T, repo *SqliteRepository, cartID string, userID string, exists bool) {
	userIDs, err := repo.Collaborators(cartID)
	if err != nil {
		t.Fatalf("Collaborators() error: %v", err)
	}

	found := false
	for _, id := range userIDs {
		if id == userID {
			found = true
			break
		}
	}
	if found != exists {
		if exists {
			t.Errorf("Expected to find collaborator %s for cart %s, but did not", userID, cartID)
		} else {
			t.Errorf("Did not expect to find collaborator %s for cart %s, but did", userID, cartID)
		}
	}
}

// A helper to print raw query output
func query(t *testing.T, db *sql.DB, format string, args ...any) {
	// print the raw output
	t.Logf("Query: "+format, args...)
	rows, err := db.Query(format, args...)
	if err != nil {
		t.Fatalf("Exec failed: %s", err)
	}
	n := 0
	for rows.Next() {
		cols, err := rows.Columns()
		n++
		if err != nil {
			t.Fatalf("Columns failed: %s", err)
		}
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			t.Fatalf("Scan failed: %s", err)
		}
		t.Logf("== ROW %d ==", n)
		for i, col := range cols {
			t.Logf("  %s: %v", col, values[i])
		}
		t.Log("== ROW END == ")
	}
	t.Logf("Total rows: %d\n\n", n)
}
