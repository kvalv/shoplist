package main

type (
	Event interface {
		IsEvent()
	}

	CartUpdated struct {
		CartID string
	}
)

func (CartUpdated) IsEvent() {}
