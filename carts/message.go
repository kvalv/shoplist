package carts

import "time"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type Message struct {
	ID        string  `db:"id"`
	CartID    string  `db:"cart_id"`
	Role      Role    `db:"role"`
	UserID    *string `db:"user_id"`
	Text      string  `db:"text"`
	ItemID    *string `db:"item_id"`
	Picture   *string `db:"picture"`
	CreatedAt time.Time `db:"created_at"`
}

func NewMessage(cartID, text string) *Message {
	return &Message{
		ID:        newID(),
		CartID:    cartID,
		Role:      RoleUser,
		Text:      text,
		CreatedAt: time.Now(),
	}
}

func (m *Message) WithUser(userID string) *Message {
	m.Role = RoleUser
	m.UserID = &userID
	return m
}

func (m *Message) WithAssistant() *Message {
	m.Role = RoleAssistant
	m.UserID = nil
	return m
}

func (m *Message) WithSystem() *Message {
	m.Role = RoleSystem
	m.UserID = nil
	return m
}

func (m *Message) WithItem(itemID string) *Message {
	m.ItemID = &itemID
	return m
}

func (m *Message) WithPicture(url string) *Message {
	m.Picture = &url
	return m
}
