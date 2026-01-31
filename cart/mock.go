package cart

import (
	"fmt"
	"slices"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type mock struct {
	data []Cart
}

// Update implements [Repository].
func (m *mock) Update(item Item) error {
	panic("unimplemented")
}

// Save implements [Repository].
func (m *mock) Save(cart *Cart) error {
	for i, c := range m.data {
		if c.ID == cart.ID {
			m.data[i] = *cart
			return nil
		}
	}
	m.data = append(m.data, *cart)
	return nil
}

var _ Repository = (*mock)(nil)

func NewMock() *mock {
	return &mock{}
}

func (m *mock) Add(CartID string, item Item) error {
	for i, cart := range m.data {
		if cart.ID == CartID {
			m.data[i].Items = append(m.data[i].Items, &item)
			return nil
		}
	}
	return nil
}

func (m *mock) New() (*Cart, error) {
	return &Cart{
		ID:        gonanoid.Must(8),
		CreatedAt: time.Now(),
	}, nil
}

func (m *mock) Updte(item Item) error {
	for _, cart := range m.data {
		for i, it := range cart.Items {
			if it.ID == item.ID {
				cart.Items[i] = &item
				return nil
			}
		}
	}
	return fmt.Errorf("item not found")
}

func (m *mock) Latest() (*Cart, error) {
	if len(m.data) == 0 {
		return nil, fmt.Errorf("no carts")
	}
	got := slices.MaxFunc(m.data, func(a, b Cart) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return &got, nil
}
