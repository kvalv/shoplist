package carts

import (
	"time"

	"github.com/kvalv/shoplist/stores"
)

type Cart struct {
	ID          string       `db:"id"`
	Name        string       `db:"name"`
	Items       []*Item      `db:"-"`
	CreatedAt   time.Time    `db:"created_at"`
	CreatedBy   *string      `db:"created_by"`
	Inactive    bool         `db:"inactive"`
	TargetStore stores.Store `db:"target_store"`
}

func (c *Cart) WithName(name string) *Cart {
	c.Name = name
	return c
}

func (c *Cart) WithCreator(userID string) *Cart {
	c.CreatedBy = &userID
	return c
}

func (c *Cart) Add(text string, userID string) *Item {
	now := time.Now()
	item := &Item{
		ID:        newID(),
		Text:      text,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: userID,
		UpdatedBy: userID,
		SortOrder: len(c.Items),
	}
	c.Items = prepend(c.Items, item)
	return item
}

func (c *Cart) Get(ID string) *Item {
	for _, item := range c.Items {
		if item.ID == ID {
			return item
		}
	}
	return nil
}

func prepend[T any](s []T, v T) []T { return append([]T{v}, s...) }
