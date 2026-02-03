package events

type (
	Event interface {
		IsEvent()
	}

	CartUpdated struct {
		CartID  string
		ItemIDs []string
	}
	CartCreated struct {
		CartID string
	}
	CartSwitched struct {
		UserID string
		CartID string
	}

	UserRegistered struct {
		UserID string
	}
)

func (CartUpdated) IsEvent()    {}
func (CartCreated) IsEvent()    {}
func (CartSwitched) IsEvent()   {}
func (UserRegistered) IsEvent() {}
