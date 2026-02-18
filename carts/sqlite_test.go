package carts

import (
	"database/sql"
	"testing"

	"github.com/kvalv/shoplist/stores/clasohlson"
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

func TestList(t *testing.T) {
	repo := NewTestRepository(t)

	// Create 3 carts with distinct names so we can verify order
	c1 := New().WithName("first")
	c2 := New().WithName("second")
	c3 := New().WithName("third")
	// Ensure ordering via incrementing timestamps
	c2.CreatedAt = c1.CreatedAt.Add(1)
	c3.CreatedAt = c1.CreatedAt.Add(2)
	for _, c := range []*Cart{c1, c2, c3} {
		repo.MustSave(c)
	}

	tests := []struct {
		name      string
		n         int
		wantNames []string
	}{
		{"all three", 5, []string{"third", "second", "first"}},
		{"limit to 2", 2, []string{"third", "second"}},
		{"limit to 1", 1, []string{"third"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.List(tt.n)
			if err != nil {
				t.Fatalf("List(%d) error: %v", tt.n, err)
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("List(%d) returned %d carts, want %d", tt.n, len(got), len(tt.wantNames))
			}
			for i, want := range tt.wantNames {
				if got[i].Name != want {
					t.Errorf("List(%d)[%d].Name = %q, want %q", tt.n, i, got[i].Name, want)
				}
			}
		})
	}
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

func TestToggleItem(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice", "bob")

	cart := New().WithCreator("alice")
	item := cart.Add("milk", "alice")
	repo.MustSave(cart)

	t.Run("toggle unchecked to checked", func(t *testing.T) {
		cartID, err := repo.ToggleItem(item.ID, "bob")
		if err != nil {
			t.Fatalf("ToggleItem() error: %v", err)
		}
		if cartID != cart.ID {
			t.Errorf("cartID = %q, want %q", cartID, cart.ID)
		}
		expectItem(t, repo, cart.ID, item.ID, func(item *Item) {
			if !item.Checked {
				t.Errorf("expected item to be checked")
			}
			if item.UpdatedBy != "bob" {
				t.Errorf("updatedBy = %q, want %q", item.UpdatedBy, "bob")
			}
		})
	})

	t.Run("toggle checked back to unchecked", func(t *testing.T) {
		_, err := repo.ToggleItem(item.ID, "alice")
		if err != nil {
			t.Fatalf("ToggleItem() error: %v", err)
		}
		expectItem(t, repo, cart.ID, item.ID, func(item *Item) {
			if item.Checked {
				t.Errorf("expected item to be unchecked")
			}
			if item.UpdatedBy != "alice" {
				t.Errorf("updatedBy = %q, want %q", item.UpdatedBy, "alice")
			}
		})
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.ToggleItem("nonexistent", "alice")
		if err == nil {
			t.Fatal("expected error for nonexistent item, got nil")
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

	if err := repo.SelectClasOhlsonItem(item.ID, "a"); err != nil {
		t.Fatalf("SelectClasOhlsonItem('a') error: %v", err)
	}

	if err := repo.SelectClasOhlsonItem(item.ID, "b"); err != nil {
		t.Fatalf("SelectClasOhlsonItem('b') error: %v", err)
	}

	if err := repo.SelectClasOhlsonItem(item.ID, "nonexistent"); err == nil {
		t.Fatalf("Expected error for nonexistent clas ID, but got none")
	}

}

func TestDeleteItem(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	item := cart.Add("skopose", "alice")
	repo.MustSave(cart)

	if err := repo.DeleteItem(item.ID); err != nil {
		t.Fatalf("DeleteItem() error: %v", err)
	}

	got, err := repo.Cart(cart.ID)
	if err != nil {
		t.Fatalf("Cart() error: %v", err)
	}
	if got.Get(item.ID) != nil {
		t.Fatalf("Expected item to be deleted, but it still exists")
	}
}

func TestDiscardItem(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	item := cart.Add("milk", "alice")
	repo.MustSave(cart)

	t.Run("basic", func(t *testing.T) {
		cartID, err := repo.DiscardItem(item.ID, "alice")
		if err != nil {
			t.Fatalf("DiscardItem() error: %v", err)
		}
		if cartID != cart.ID {
			t.Errorf("cartID = %q, want %q", cartID, cart.ID)
		}
		expectItem(t, repo, cart.ID, item.ID, func(item *Item) {
			if !item.Discarded {
				t.Errorf("expected item to be discarded")
			}
		})
	})

	t.Run("idempotent", func(t *testing.T) {
		_, err := repo.DiscardItem(item.ID, "alice")
		if err != nil {
			t.Fatalf("second DiscardItem() error: %v", err)
		}
		expectItem(t, repo, cart.ID, item.ID, func(item *Item) {
			if !item.Discarded {
				t.Errorf("expected item to still be discarded")
			}
		})
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.DiscardItem("nonexistent", "alice")
		if err == nil {
			t.Fatal("expected error for nonexistent item, got nil")
		}
	})
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

func TestMessage(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	item := cart.Add("milk", "alice")
	repo.MustSave(cart)

	t.Run("basic", func(t *testing.T) {
		msg := NewMessage(cart.ID, "don't forget the oat milk").WithUser("alice")
		if err := repo.AddMessage(msg); err != nil {
			t.Fatalf("AddMessage() error: %v", err)
		}

		msgs, err := repo.Messages(cart.ID)
		if err != nil {
			t.Fatalf("Messages() error: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		got := msgs[0]
		if got.Text != "don't forget the oat milk" {
			t.Errorf("text = %q, want %q", got.Text, "don't forget the oat milk")
		}
		if got.Role != RoleUser {
			t.Errorf("role = %q, want %q", got.Role, RoleUser)
		}
		if got.UserID == nil || *got.UserID != "alice" {
			t.Errorf("user_id = %v, want 'alice'", got.UserID)
		}
		if got.ItemID != nil {
			t.Errorf("item_id = %v, want nil", got.ItemID)
		}
	})

	t.Run("with item link", func(t *testing.T) {
		msg := NewMessage(cart.ID, "this one is expired").WithUser("alice").WithItem(item.ID)
		if err := repo.AddMessage(msg); err != nil {
			t.Fatalf("AddMessage() error: %v", err)
		}

		msgs, err := repo.Messages(cart.ID)
		if err != nil {
			t.Fatalf("Messages() error: %v", err)
		}
		// find our message
		var got *Message
		for _, m := range msgs {
			if m.ID == msg.ID {
				got = m
			}
		}
		if got == nil {
			t.Fatalf("message %s not found", msg.ID)
		}
		if got.ItemID == nil || *got.ItemID != item.ID {
			t.Errorf("item_id = %v, want %q", got.ItemID, item.ID)
		}
	})

	t.Run("survives item delete", func(t *testing.T) {
		item2 := cart.Add("bread", "alice")
		repo.MustSave(cart)

		msg := NewMessage(cart.ID, "sold out").WithSystem().WithItem(item2.ID)
		if err := repo.AddMessage(msg); err != nil {
			t.Fatalf("AddMessage() error: %v", err)
		}

		if err := repo.DeleteItem(item2.ID); err != nil {
			t.Fatalf("DeleteItem() error: %v", err)
		}

		msgs, err := repo.Messages(cart.ID)
		if err != nil {
			t.Fatalf("Messages() error: %v", err)
		}
		var got *Message
		for _, m := range msgs {
			if m.ID == msg.ID {
				got = m
			}
		}
		if got == nil {
			t.Fatalf("message %s not found after item delete", msg.ID)
		}
		if got.ItemID != nil {
			t.Errorf("item_id = %v, want nil after item delete", got.ItemID)
		}
	})

	t.Run("ordered by time", func(t *testing.T) {
		cart2 := New()
		repo.MustSave(cart2)

		m1 := NewMessage(cart2.ID, "first").WithUser("alice")
		m2 := NewMessage(cart2.ID, "second").WithAssistant()
		m3 := NewMessage(cart2.ID, "third").WithSystem()
		// ensure ordering via incrementing timestamps
		m2.CreatedAt = m1.CreatedAt.Add(1)
		m3.CreatedAt = m1.CreatedAt.Add(2)

		for _, m := range []*Message{m3, m1, m2} { // insert out of order
			if err := repo.AddMessage(m); err != nil {
				t.Fatalf("AddMessage() error: %v", err)
			}
		}

		msgs, err := repo.Messages(cart2.ID)
		if err != nil {
			t.Fatalf("Messages() error: %v", err)
		}
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
		if msgs[0].Text != "first" || msgs[1].Text != "second" || msgs[2].Text != "third" {
			t.Errorf("order = [%s, %s, %s], want [first, second, third]", msgs[0].Text, msgs[1].Text, msgs[2].Text)
		}
	})

	t.Run("roles", func(t *testing.T) {
		cart3 := New()
		repo.MustSave(cart3)

		userMsg := NewMessage(cart3.ID, "hey").WithUser("alice")
		assistantMsg := NewMessage(cart3.ID, "hello").WithAssistant()
		systemMsg := NewMessage(cart3.ID, "item added").WithSystem()

		for _, m := range []*Message{userMsg, assistantMsg, systemMsg} {
			if err := repo.AddMessage(m); err != nil {
				t.Fatalf("AddMessage() error: %v", err)
			}
		}

		msgs, err := repo.Messages(cart3.ID)
		if err != nil {
			t.Fatalf("Messages() error: %v", err)
		}
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}

		for _, m := range msgs {
			switch m.Role {
			case RoleUser:
				if m.UserID == nil || *m.UserID != "alice" {
					t.Errorf("user role: user_id = %v, want 'alice'", m.UserID)
				}
			case RoleAssistant, RoleSystem:
				if m.UserID != nil {
					t.Errorf("%s role: user_id = %v, want nil", m.Role, m.UserID)
				}
			default:
				t.Errorf("unexpected role %q", m.Role)
			}
		}
	})
}

func TestUnseenMessageCount(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice", "bob")

	cart := New().WithCreator("alice")
	repo.MustSave(cart)
	repo.AddCollaborators(cart.ID, "bob")

	t.Run("no messages", func(t *testing.T) {
		count, err := repo.UnseenMessageCount(cart.ID, "alice")
		if err != nil {
			t.Fatalf("UnseenMessageCount() error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 unseen, got %d", count)
		}
	})

	t.Run("all unseen when chat_seen_at is NULL", func(t *testing.T) {
		m1 := NewMessage(cart.ID, "hey alice").WithUser("bob")
		m2 := NewMessage(cart.ID, "second msg").WithUser("bob")
		for _, m := range []*Message{m1, m2} {
			if err := repo.AddMessage(m); err != nil {
				t.Fatalf("AddMessage() error: %v", err)
			}
		}

		count, err := repo.UnseenMessageCount(cart.ID, "alice")
		if err != nil {
			t.Fatalf("UnseenMessageCount() error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 unseen, got %d", count)
		}
	})

	t.Run("own messages excluded", func(t *testing.T) {
		count, err := repo.UnseenMessageCount(cart.ID, "bob")
		if err != nil {
			t.Fatalf("UnseenMessageCount() error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 unseen (own messages), got %d", count)
		}
	})

	t.Run("after marking seen, count resets", func(t *testing.T) {
		if err := repo.UpdateChatSeenAt(cart.ID, "alice"); err != nil {
			t.Fatalf("UpdateChatSeenAt() error: %v", err)
		}
		count, err := repo.UnseenMessageCount(cart.ID, "alice")
		if err != nil {
			t.Fatalf("UnseenMessageCount() error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 after marking seen, got %d", count)
		}
	})

	t.Run("new message after seen increments count", func(t *testing.T) {
		m3 := NewMessage(cart.ID, "new msg").WithUser("bob")
		if err := repo.AddMessage(m3); err != nil {
			t.Fatalf("AddMessage() error: %v", err)
		}
		count, err := repo.UnseenMessageCount(cart.ID, "alice")
		if err != nil {
			t.Fatalf("UnseenMessageCount() error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 unseen after new message, got %d", count)
		}
	})
}

func TestSortOrder(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	a := cart.Add("first", "alice")
	b := cart.Add("second", "alice")
	c := cart.Add("third", "alice")
	repo.MustSave(cart)

	t.Run("auto-assigned on add", func(t *testing.T) {
		if a.SortOrder != 0 {
			t.Errorf("first item sort_order = %d, want 0", a.SortOrder)
		}
		if b.SortOrder != 1 {
			t.Errorf("second item sort_order = %d, want 1", b.SortOrder)
		}
		if c.SortOrder != 2 {
			t.Errorf("third item sort_order = %d, want 2", c.SortOrder)
		}
	})

	t.Run("persisted and loaded in order", func(t *testing.T) {
		got, err := repo.Cart(cart.ID)
		if err != nil {
			t.Fatalf("Cart() error: %v", err)
		}
		if len(got.Items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(got.Items))
		}
		wantTexts := []string{"first", "second", "third"}
		for i, want := range wantTexts {
			if got.Items[i].Text != want {
				t.Errorf("items[%d].Text = %q, want %q", i, got.Items[i].Text, want)
			}
			if got.Items[i].SortOrder != i {
				t.Errorf("items[%d].SortOrder = %d, want %d", i, got.Items[i].SortOrder, i)
			}
		}
	})

	t.Run("reorder persists", func(t *testing.T) {
		// swap first and third
		got, _ := repo.Cart(cart.ID)
		got.Items[0].SortOrder = 2
		got.Items[2].SortOrder = 0
		repo.MustSave(got)

		got2, err := repo.Cart(cart.ID)
		if err != nil {
			t.Fatalf("Cart() error: %v", err)
		}
		wantTexts := []string{"third", "second", "first"}
		for i, want := range wantTexts {
			if got2.Items[i].Text != want {
				t.Errorf("after reorder: items[%d].Text = %q, want %q", i, got2.Items[i].Text, want)
			}
		}
	})
}

func TestReorderItems(t *testing.T) {
	repo := NewTestRepository(t).WithUsers("alice")

	cart := New()
	a := cart.Add("unchecked", "alice")
	b := cart.Add("checked", "alice")
	c := cart.Add("discarded", "alice")
	d := cart.Add("also-unchecked", "alice")
	b.Checked = true
	c.Discarded = true
	repo.MustSave(cart)

	if err := repo.ReorderItems(cart.ID); err != nil {
		t.Fatalf("ReorderItems() error: %v", err)
	}

	got, err := repo.Cart(cart.ID)
	if err != nil {
		t.Fatalf("Cart() error: %v", err)
	}

	// Expected order: unchecked (by created_at), checked, discarded
	wantIDs := []string{a.ID, d.ID, b.ID, c.ID}
	for i, wantID := range wantIDs {
		if got.Items[i].ID != wantID {
			t.Errorf("items[%d].ID = %q (%s), want %q", i, got.Items[i].ID, got.Items[i].Text, wantID)
		}
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

