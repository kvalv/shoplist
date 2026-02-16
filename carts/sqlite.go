package carts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/kvalv/shoplist/stores/clasohlson"
)

type SqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*SqliteRepository, error) {
	return &SqliteRepository{db: db}, nil
}

func New() *Cart {
	return &Cart{
		ID:        newID(),
		CreatedAt: time.Now(),
	}
}

func (r *SqliteRepository) Save(cart *Cart) error {
	_, err := r.db.Exec(
		`INSERT INTO carts (id, name, created_at, created_by, target_store, inactive) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name = excluded.name, target_store = excluded.target_store, inactive = excluded.inactive`,
		cart.ID, cart.Name, cart.CreatedAt, cart.CreatedBy, cart.TargetStore, cart.Inactive,
	)

	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	for _, item := range cart.Items {
		if err := r.saveItem(cart.ID, item); err != nil {
			return fmt.Errorf("items.save: %w", err)
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

	var chosen *string
	if item.Clas != nil && item.Clas.Chosen != "" {
		chosen = &item.Clas.Chosen
	}
	_, err = tx.Exec(
		`INSERT INTO items (id, cart_id, text, checked, created_at, updated_at, created_by, updated_by, clas_chosen, discarded) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET checked = excluded.checked, updated_at = excluded.updated_at, updated_by = excluded.updated_by, clas_chosen = excluded.clas_chosen, discarded = excluded.discarded`,
		item.ID, cartID, item.Text, item.Checked, item.CreatedAt, item.UpdatedAt, item.CreatedBy, item.UpdatedBy, chosen, item.Discarded,
	)

	if err != nil {
		return fmt.Errorf("saveItem: %w", err)
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
				`INSERT INTO clas_candidates (item_id, idx, clas_id, name, price, url, picture, stock, area, shelf)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				item.ID, i, c.ID, c.Name, c.Price, c.URL, c.Picture, c.Stock, area, shelf,
			)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *SqliteRepository) Latest() (*Cart, error) {
	var cart Cart
	if err := get(r.db, &cart, `SELECT id, name, created_at, target_store, inactive FROM carts ORDER BY created_at DESC LIMIT 1`); err != nil {
		return nil, err
	}
	return r.loadCartItems(&cart)
}

func (r *SqliteRepository) List(n int) ([]*Cart, error) {
	var carts []*Cart
	if err := many(&carts, r.db, `SELECT id, name, created_at, target_store, inactive FROM carts ORDER BY created_at DESC LIMIT ?`, n); err != nil {
		return nil, err
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
	var cart Cart
	if err := get(r.db, &cart, `SELECT id, name, created_at, target_store, inactive FROM carts WHERE id = ?`, ID); err != nil {
		return nil, err
	}
	return r.loadCartItems(&cart)
}

func (r *SqliteRepository) loadCartItems(cart *Cart) (*Cart, error) {
	if err := many(&cart.Items, r.db, `SELECT id, text, checked, created_at, updated_at, clas_chosen, created_by, updated_by, discarded FROM items WHERE cart_id = ? ORDER BY created_at ASC`, cart.ID); err != nil {
		return nil, err
	}

	for _, item := range cart.Items {
		if item.ClasChosen != nil {
			item.Clas = &ClasSearch{Chosen: *item.ClasChosen}
		}
		if err := r.loadClasCandidates(item); err != nil {
			return nil, err
		}
	}
	return cart, nil
}

type clasCandidateRow struct {
	ClasID  string  `db:"clas_id"`
	Name    string  `db:"name"`
	Price   float64 `db:"price"`
	URL     string  `db:"url"`
	Picture string  `db:"picture"`
	Stock   int     `db:"stock"`
	Area    *string `db:"area"`
	Shelf   *string `db:"shelf"`
}

func (r *SqliteRepository) loadClasCandidates(item *Item) error {
	var rows []clasCandidateRow
	if err := many(&rows, r.db, `SELECT clas_id, name, price, url, picture, stock, area, shelf FROM clas_candidates WHERE item_id = ? ORDER BY idx`, item.ID); err != nil {
		return err
	}

	for _, row := range rows {
		c := clasohlson.Item{
			ID:      row.ClasID,
			Name:    row.Name,
			Price:   row.Price,
			URL:     row.URL,
			Picture: row.Picture,
			Stock:   row.Stock,
		}
		if row.Area != nil && row.Shelf != nil {
			c.Locations = []clasohlson.ShelfLocation{{Area: *row.Area, Shelf: *row.Shelf}}
		}
		if item.Clas == nil {
			item.Clas = &ClasSearch{}
		}
		item.Clas.Candidates = append(item.Clas.Candidates, c)
	}
	return nil
}

func (r *SqliteRepository) SelectClasOhlsonItem(itemID string, clasID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var res struct {
		Count int
	}

	if err := get(tx, &res, `SELECT count(*) as count FROM clas_candidates WHERE item_id = ? AND clas_id = ?`, itemID, clasID); err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	if res.Count == 0 {
		return fmt.Errorf("clas item %q not found for item %s", clasID, itemID)
	}

	if _, err := tx.Exec("update items set clas_chosen = ? where id = ?", clasID, itemID); err != nil {
		return fmt.Errorf("update error: %w", err)
	}

	return tx.Commit()
}

func (r *SqliteRepository) AddMessage(msg *Message) error {
	_, err := r.db.Exec(
		`INSERT INTO messages (id, cart_id, role, user_id, text, item_id, picture, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.CartID, msg.Role, msg.UserID, msg.Text, msg.ItemID, msg.Picture, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (r *SqliteRepository) Message(id string) (*Message, error) {
	var msg Message
	if err := get(r.db, &msg, `SELECT id, cart_id, role, user_id, text, item_id, picture, created_at FROM messages WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *SqliteRepository) Messages(cartID string) ([]*Message, error) {
	var msgs []*Message
	if err := many(&msgs, r.db, `SELECT id, cart_id, role, user_id, text, item_id, picture, created_at FROM messages WHERE cart_id = ? ORDER BY created_at ASC`, cartID); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *SqliteRepository) DeleteItem(itemID string) error {
	_, err := r.db.Exec("DELETE FROM items WHERE id = ?", itemID)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

func (r *SqliteRepository) Collaborators(cartID string) ([]string, error) {
	var rows []struct {
		UserID string `db:"user_id"`
	}
	if err := many(&rows, r.db, `SELECT user_id FROM collaborators WHERE cart_id = ?`, cartID); err != nil {
		return nil, err
	}
	users := make([]string, len(rows))
	for i, row := range rows {
		users[i] = row.UserID
	}
	return users, nil
}

func (r *SqliteRepository) AddCollaborators(cartID string, userIDs ...string) error {
	for _, userID := range userIDs {
		if _, err := r.db.Exec(
			`INSERT INTO collaborators (cart_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
			cartID, userID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *SqliteRepository) ToggleItem(itemID, userID string) (cartID string, err error) {
	var row struct {
		CartID string `db:"cart_id"`
	}
	if err := get(r.db, &row, `
		UPDATE items SET checked = NOT checked, updated_at = ?, updated_by = ?
		WHERE id = ?
		RETURNING cart_id`,
		time.Now(), userID, itemID,
	); err != nil {
		return "", fmt.Errorf("toggle item: %w", err)
	}
	return row.CartID, nil
}

func (r *SqliteRepository) DiscardItem(itemID, userID string) (cartID string, err error) {
	var row struct {
		CartID string `db:"cart_id"`
	}
	if err := get(r.db, &row, `
		UPDATE items SET discarded = true, updated_at = ?, updated_by = ?
		WHERE id = ?
		RETURNING cart_id`,
		time.Now(), userID, itemID,
	); err != nil {
		return "", fmt.Errorf("discard item: %w", err)
	}
	return row.CartID, nil
}


func (r *SqliteRepository) UpdateChatSeenAt(cartID, userID string) error {
	_, err := r.db.Exec(
		`UPDATE collaborators SET chat_seen_at = ? WHERE cart_id = ? AND user_id = ?`,
		time.Now(), cartID, userID,
	)
	if err != nil {
		return fmt.Errorf("update chat_seen_at: %w", err)
	}
	return nil
}

func (r *SqliteRepository) UnseenMessageCount(cartID, userID string) (int, error) {
	var result struct {
		Count int `db:"count"`
	}
	if err := get(r.db, &result, `
		SELECT COUNT(*) as count FROM messages m
		JOIN collaborators c ON c.cart_id = m.cart_id AND c.user_id = ?
		WHERE m.cart_id = ?
		  AND (m.user_id IS NULL OR m.user_id != ?)
		  AND (c.chat_seen_at IS NULL OR m.created_at > c.chat_seen_at)
	`, userID, cartID, userID); err != nil {
		return 0, fmt.Errorf("unseen message count: %w", err)
	}
	return result.Count, nil
}

func get(db sqlscan.Querier, dest any, query string, args ...any) error {
	if err := sqlscan.Get(context.TODO(), db, dest, query, args...); err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	return nil
}
func many(dest any, db sqlscan.Querier, query string, args ...any) error {
	if err := sqlscan.Select(context.TODO(), db, dest, query, args...); err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	return nil
}
