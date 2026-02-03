package carts

import (
	"database/sql"
	"testing"

	"github.com/kvalv/shoplist/migrations"
	_ "modernc.org/sqlite"
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

func TestAddAndTick(t *testing.T) {
	repo, _ := NewMock()

	cart := New()
	item := cart.Add("milk", "alice")

	if err := repo.Save(cart); err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	item.Toggle("bob")

	if err := repo.Save(cart); err != nil {
		t.Fatalf("failed to tick: %v", err)
	}

	expectItem(t, repo, cart.ID, item.ID, func(item *Item) {
		if !item.Checked {
			t.Errorf("Expected item to be checked, but it was not")
		}
		if item.CreatedBy != "alice" {
			t.Errorf("Expected item to be created by 'alice', but got '%s'", item.CreatedBy)
		}
		if item.UpdatedBy != "bob" {
			t.Errorf("Expected item to be updated by 'bob', but got '%s'", item.UpdatedBy)
		}
	})
}

func expectItem(t *testing.T, repo *SqliteRepository, cartID string, itemID string, cb func(item *Item)) {
	cart, err := repo.Cart(cartID)
	if err != nil {
		t.Fatalf("Cart() error: %v", err)
	}
	item := cart.Get(itemID)
	if item == nil {
		t.Fatalf("Item %s not found in cart %s", itemID, cartID)
	}
	cb(item)
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

func NewMock(dsn ...string) (*SqliteRepository, *sql.DB) {
	dsn_ := ":memory:"
	if len(dsn) > 0 {
		dsn_ = dsn[0]
	}

	db, err := sql.Open("sqlite", dsn_)
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		panic(err)
	}
	if err := migrations.Migrate(db); err != nil {
		panic(err)
	}

	mustWithUsers(db, "alice", "bob")

	repo, err := NewRepository(db)
	if err != nil {
		panic(err)
	}
	return repo, db
}

func mustWithUsers(db *sql.DB, userIDs ...string) {
	for _, userID := range userIDs {
		if _, err := db.Exec(`INSERT INTO users (user_id, name, email) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, userID, userID, userID+"@example.com"); err != nil {
			panic(err)
		}
	}
}
