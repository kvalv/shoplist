package carts

import (
	"database/sql"
	"testing"

	"github.com/kvalv/shoplist/migrations"
	_ "modernc.org/sqlite"
)

// NewTestRepository creates an in-memory SQLite repository for testing.
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

func (r *TestRepository) Close() {
	r.db.Close()
}

func (r *TestRepository) WithUsers(userIDs ...string) *TestRepository {
	for _, userID := range userIDs {
		if _, err := r.db.Exec(`INSERT INTO users (user_id, name, email) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, userID, userID, userID+"@example.com"); err != nil {
			r.t.Fatalf("WithUsers() error: %v", err)
		}
	}
	return r
}
