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
	ChatAdded struct {
		CartID    string
		MessageID string
	}
	ChatOpened struct {
		CartID string
		UserID string
	}
	ItemDiscarded struct {
		CartID string
		ItemID string
		UserID string
		Reason string
	}
)

func (CartUpdated) IsEvent()    {}
func (CartCreated) IsEvent()    {}
func (CartSwitched) IsEvent()   {}
func (UserRegistered) IsEvent() {}
func (ChatAdded) IsEvent()      {}
func (ChatOpened) IsEvent()     {}
func (ItemDiscarded) IsEvent()  {}
