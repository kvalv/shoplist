package carts

import (
	"time"

	"github.com/kvalv/shoplist/stores/clasohlson"
)

type Item struct {
	ID         string    `db:"id"`
	Text       string    `db:"text"`
	Checked    bool      `db:"checked"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	UpdatedBy  string    `db:"updated_by"`
	CreatedBy  string    `db:"created_by"`
	ClasChosen *string   `db:"clas_chosen"`
	Discarded  bool      `db:"discarded"`
	SortOrder  int       `db:"sort_order"`

	Clas *ClasSearch `db:"-"`
}

func (i *Item) Toggle(toggledBy string) *Item {
	i.Checked = !i.Checked
	i.UpdatedAt = time.Now()
	i.UpdatedBy = toggledBy
	return i
}

type ClasSearch struct {
	Candidates []clasohlson.Item
	Chosen     string // clas item ID, empty when not set
}

// Selected returns the chosen item, or nil if none selected
func (c *ClasSearch) Selected() *clasohlson.Item {
	if c == nil || c.Chosen == "" {
		return nil
	}
	for i := range c.Candidates {
		if c.Candidates[i].ID == c.Chosen {
			return &c.Candidates[i]
		}
	}
	return nil
}
