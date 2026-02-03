package carts

import (
	"time"

	"github.com/kvalv/shoplist/stores/clasohlson"
)

type Item struct {
	ID        string
	Text      string
	Checked   bool
	CreatedAt time.Time
	UpdatedAt time.Time

	UpdatedBy string
	CreatedBy string

	Clas *ClasSearch
}

func (i *Item) Toggle(toggledBy string) *Item {
	i.Checked = !i.Checked
	i.UpdatedAt = time.Now()
	i.UpdatedBy = toggledBy
	return i
}

type ClasSearch struct {
	Candidates []clasohlson.Item
	Chosen     *int // index into Candidates
}

// Selected returns the chosen item, or nil if none selected
func (c *ClasSearch) Selected() *clasohlson.Item {
	if c == nil || c.Chosen == nil || *c.Chosen < 0 || *c.Chosen >= len(c.Candidates) {
		return nil
	}
	return &c.Candidates[*c.Chosen]
}
