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

	UserRegistered struct {
		UserID string
	}
)

func (CartUpdated) IsEvent()    {}
func (CartSwitched) IsEvent()   {}
func (UserRegistered) IsEvent() {}
