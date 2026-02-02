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

	Clas *ClasSearch
}

func (i *Item) SetChecked(value ...bool) {
	if len(value) > 0 {
		i.Checked = value[0]
	} else {
		i.Checked = true
	}
	i.UpdatedAt = time.Now()
}
func (i *Item) Toggle() {
	i.Checked = !i.Checked
	i.UpdatedAt = time.Now()
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
