package carts

import (
	"database/sql"
	"testing"

	"github.com/kvalv/shoplist/migrations"
	"github.com/kvalv/shoplist/stores/clasohlson"
	_ "modernc.org/sqlite"
)

func TestSqliteBasic(t *testing.T) {
	repo := NewTestRepository(t)

	// Create a cart
	cart := New()
	if err := repo.Save(cart); err != nil {
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
	repo := NewTestRepository(t).WithUsers("alice", "bob")

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

func TestSelectClasOhlsonItem(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	item := cart.Add("milk", "alice")
	item.Clas = &ClasSearch{
		Candidates: []clasohlson.Item{
			{
				ID:        "a",
				Name:      "Skopose",
				Price:     100,
				URL:       "google.com",
				Picture:   "gogle.com",
				Reviews:   1,
				Stock:     1,
				Locations: []clasohlson.ShelfLocation{},
			},
			{
				ID:        "b",
				Name:      "Enda en skopose",
				Price:     100,
				URL:       "google.com",
				Picture:   "gogle.com",
				Reviews:   1,
				Stock:     5,
				Locations: []clasohlson.ShelfLocation{},
			},
		},
	}

	repo.MustSave(cart)

	if err := repo.SelectClasOhlsonItem(item.ID, 0); err != nil {
		t.Fatalf("SelectClasOhlsonItem(0) error: %v", err)
	}

	if err := repo.SelectClasOhlsonItem(item.ID, 1); err != nil {
		t.Fatalf("SelectClasOhlsonItem(1) error: %v", err)
	}

	if err := repo.SelectClasOhlsonItem(item.ID, 2); err == nil {
		t.Fatalf("Expected error for out-of-bounds index 2, but got none")
	}

}

func expectItem(t *testing.T, repo *TestRepository, cartID string, itemID string, cb func(item *Item)) {
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
	repo := NewTestRepository(t).WithUsers("alice", "bob")

	t.Run("no collaborator", func(t *testing.T) {
		cart := New()
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "foo", false)
	})

	t.Run("with collaborator due to trigger", func(t *testing.T) {
		cart := New().WithCreator("alice")
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "alice", true)
	})

	t.Run("add collaborator", func(t *testing.T) {
		cart := New()
		if err := repo.Save(cart); err != nil {
			t.Fatalf("Failed to save cart: %s", err)
		}
		expectCollaborator(t, repo, cart.ID, "bob", false)

		if err := repo.AddCollaborators(cart.ID, "bob"); err != nil {
			t.Fatalf("AddCollaborator() error: %v", err)
		}
		expectCollaborator(t, repo, cart.ID, "bob", true)
	})
}

func expectCollaborator(t *testing.T, repo *TestRepository, cartID string, userID string, exists bool) {
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

func NewTestRepository(t *testing.T) *TestRepository {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	if err := migrations.Migrate(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo, err := NewRepository(db)
	if err != nil {
		panic(err)
	}
	return &TestRepository{SqliteRepository: *repo, t: t, db: db}
}

// A SqliteRepository, wrapped with utility funcs
type TestRepository struct {
	SqliteRepository
	t  *testing.T
	db *sql.DB
}

func (r *TestRepository) MustSave(cart *Cart) *TestRepository {
	if err := r.Save(cart); err != nil {
		r.t.Fatalf("Save() error: %v", err)
	}
	return r
}

func (r *TestRepository) WithUsers(userIDs ...string) *TestRepository {
	for _, userID := range userIDs {
		if _, err := r.db.Exec(`INSERT INTO users (user_id, name, email) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, userID, userID, userID+"@example.com"); err != nil {
			r.t.Fatalf("WithUsers() error: %v", err)
		}
	}
	return r
}
