package cart

import (
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Cart struct {
	ID        string
	Items     []*Item
	CreatedAt time.Time
}

func (c *Cart) Add(text string) *Item {
	item := &Item{
		ID:   gonanoid.Must(8),
		Text: text,
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
