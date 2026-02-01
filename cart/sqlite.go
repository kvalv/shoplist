package cart

import (
	"database/sql"
	"time"

	"github.com/kvalv/shoplist/stores/clasohlson"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type SqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*SqliteRepository, error) {
	repo := &SqliteRepository{db: db}
	if err := repo.Migrate(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *SqliteRepository) Migrate() error {
	stmts := []string{
		`create table if not exists carts (
			id text primary key,
			name text not null default '',
			created_at datetime not null,
			target_store integer not null,
			inactive boolean not null default false
		)`,
		`create table if not exists items (
			id text primary key,
			cart_id text not null references carts(id) on delete cascade,
			text text not null,
			checked boolean not null,
			created_at datetime not null,
			updated_at datetime,
			clas_chosen integer,
			unique(cart_id, text)
		)`,
		`create table if not exists clas_candidates (
			item_id text not null references items(id) on delete cascade,
			idx integer not null,
			gtm_id text not null,
			name text not null,
			price real not null,
			url text not null,
			picture text not null,
			reviews integer not null,
			stock integer not null,
			area text,
			shelf text,
			primary key(item_id, idx)
		)`,
	}

	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *SqliteRepository) New() (*Cart, error) {
	cart := &Cart{
		ID:        gonanoid.Must(8),
		CreatedAt: time.Now(),
	}
	_, err := r.db.Exec(
		`INSERT INTO carts (id, name, created_at, target_store, inactive) VALUES (?, ?, ?, ?, ?)`,
		cart.ID, cart.Name, cart.CreatedAt, cart.TargetStore, cart.Inactive,
	)
	return cart, err
}

func (r *SqliteRepository) Save(cart *Cart) error {
	_, err := r.db.Exec(
		`INSERT INTO carts (id, name, created_at, target_store, inactive) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name = excluded.name, target_store = excluded.target_store, inactive = excluded.inactive`,
		cart.ID, cart.Name, cart.CreatedAt, cart.TargetStore, cart.Inactive,
	)
	if err != nil {
		return err
	}

	for _, item := range cart.Items {
		if err := r.saveItem(cart.ID, item); err != nil {
			return err
		}
	}
	return nil
}

func (r *SqliteRepository) saveItem(cartID string, item *Item) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var chosen *int
	if item.Clas != nil {
		chosen = item.Clas.Chosen
	}
	_, err = tx.Exec(
		`INSERT INTO items (id, cart_id, text, checked, created_at, updated_at, clas_chosen) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET checked = excluded.checked, updated_at = excluded.updated_at, clas_chosen = excluded.clas_chosen`,
		item.ID, cartID, item.Text, item.Checked, item.CreatedAt, item.UpdatedAt, chosen,
	)
	if err != nil {
		return err
	}

	if item.Clas != nil && len(item.Clas.Candidates) > 0 {
		tx.Exec(`DELETE FROM clas_candidates WHERE item_id = ?`, item.ID)
		for i, c := range item.Clas.Candidates {
			var area, shelf *string
			if len(c.Locations) > 0 {
				area = &c.Locations[0].Area
				shelf = &c.Locations[0].Shelf
			}
			_, err := tx.Exec(
				`INSERT INTO clas_candidates (item_id, idx, gtm_id, name, price, url, picture, reviews, stock, area, shelf)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				item.ID, i, c.ID, c.Name, c.Price, c.URL, c.Picture, c.Reviews, c.Stock, area, shelf,
			)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *SqliteRepository) Update(item *Item) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var chosen *int
	if item.Clas != nil {
		chosen = item.Clas.Chosen
	}
	_, err = tx.Exec(
		`UPDATE items SET text = ?, checked = ?, updated_at = ?, clas_chosen = ? WHERE id = ?`,
		item.Text, item.Checked, item.UpdatedAt, chosen, item.ID,
	)
	if err != nil {
		return err
	}

	tx.Exec(`DELETE FROM clas_candidates WHERE item_id = ?`, item.ID)
	if item.Clas != nil {
		for i, c := range item.Clas.Candidates {
			var area, shelf *string
			if len(c.Locations) > 0 {
				area = &c.Locations[0].Area
				shelf = &c.Locations[0].Shelf
			}
			_, err := tx.Exec(
				`INSERT INTO clas_candidates (item_id, idx, gtm_id, name, price, url, picture, reviews, stock, area, shelf)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				item.ID, i, c.ID, c.Name, c.Price, c.URL, c.Picture, c.Reviews, c.Stock, area, shelf,
			)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *SqliteRepository) Latest() (*Cart, error) {
	row := r.db.QueryRow(`SELECT id, name, created_at, target_store, inactive FROM carts ORDER BY created_at DESC LIMIT 1`)
	cart := &Cart{}
	if err := row.Scan(&cart.ID, &cart.Name, &cart.CreatedAt, &cart.TargetStore, &cart.Inactive); err != nil {
		return nil, err
	}
	return r.loadCartItems(cart)
}

func (r *SqliteRepository) List(n int) ([]*Cart, error) {
	rows, err := r.db.Query(`SELECT id, name, created_at, target_store, inactive FROM carts ORDER BY created_at DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carts []*Cart
	for rows.Next() {
		cart := &Cart{}
		if err := rows.Scan(&cart.ID, &cart.Name, &cart.CreatedAt, &cart.TargetStore, &cart.Inactive); err != nil {
			return nil, err
		}
		carts = append(carts, cart)
	}

	for i, cart := range carts {
		cart, err := r.loadCartItems(cart)
		if err != nil {
			return nil, err
		}
		carts[i] = cart
	}
	return carts, nil
}

func (r *SqliteRepository) Cart(ID string) (*Cart, error) {
	row := r.db.QueryRow(`SELECT id, name, created_at, target_store, inactive FROM carts WHERE id = ?`, ID)
	cart := &Cart{}
	if err := row.Scan(&cart.ID, &cart.Name, &cart.CreatedAt, &cart.TargetStore, &cart.Inactive); err != nil {
		return nil, err
	}
	return r.loadCartItems(cart)
}

func (r *SqliteRepository) loadCartItems(cart *Cart) (*Cart, error) {
	rows, err := r.db.Query(`SELECT id, text, checked, created_at, updated_at, clas_chosen FROM items WHERE cart_id = ? ORDER BY checked ASC, updated_at DESC`, cart.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := &Item{}
		var chosen *int
		var updatedAt *time.Time
		if err := rows.Scan(&item.ID, &item.Text, &item.Checked, &item.CreatedAt, &updatedAt, &chosen); err != nil {
			return nil, err
		}
		if updatedAt != nil {
			item.UpdatedAt = *updatedAt
		} else {
			item.UpdatedAt = item.CreatedAt
		}
		if chosen != nil {
			item.Clas = &ClasSearch{Chosen: chosen}
		}
		cart.Items = append(cart.Items, item)
	}

	for _, item := range cart.Items {
		if err := r.loadClasCandidates(item); err != nil {
			return nil, err
		}
	}
	return cart, nil
}

func (r *SqliteRepository) loadClasCandidates(item *Item) error {
	rows, err := r.db.Query(
		`SELECT gtm_id, name, price, url, picture, reviews, stock, area, shelf
		 FROM clas_candidates WHERE item_id = ? ORDER BY idx`, item.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var c clasohlson.Item
		var area, shelf *string
		if err := rows.Scan(&c.ID, &c.Name, &c.Price, &c.URL, &c.Picture, &c.Reviews, &c.Stock, &area, &shelf); err != nil {
			return err
		}
		if area != nil && shelf != nil {
			c.Locations = []clasohlson.ShelfLocation{{Area: *area, Shelf: *shelf}}
		}
		if item.Clas == nil {
			item.Clas = &ClasSearch{}
		}
		item.Clas.Candidates = append(item.Clas.Candidates, c)
	}
	return nil
}
