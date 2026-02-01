package events

type (
	Event interface {
		IsEvent()
	}

	CartUpdated struct {
		CartID  string
		ItemIDs []string
	}
	CartSwitched struct {
		CartID string
	}
)

func (CartUpdated) IsEvent()  {}
func (CartSwitched) IsEvent() {}
