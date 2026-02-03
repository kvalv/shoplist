package carts

import (
	"time"

	"github.com/kvalv/shoplist/stores"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Cart struct {
	ID        string
	Name      string
	Items     []*Item
	CreatedAt time.Time
	CreatedBy *string

	// Inactive is true when there is at least one item ticked off and the
	// last tick happened more than 1 day ago. Computed by a background worker.
	Inactive bool

	// Business logic related to clas ohlson is different than kiwi.
	TargetStore stores.Store
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
		ID:        gonanoid.Must(8),
		Text:      text,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: userID,
		UpdatedBy: userID,
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
