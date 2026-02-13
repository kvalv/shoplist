# Chat / Notes Feature — Backend Plan

## Overview
A chat-like feed attached to a cart where users, an LLM assistant, or the system
can post notes. Each message can optionally reference one item in the cart.

## Domain Model

```go
// carts/message.go

type Message struct {
    ID        string    `db:"id"`
    CartID    string    `db:"cart_id"`
    Role      string    `db:"role"`       // "user", "assistant", "system"
    UserID    *string   `db:"user_id"`    // set for role="user", nil for others
    Text      string    `db:"text"`       // the message body
    ItemID    *string   `db:"item_id"`    // optional link to one item
    CreatedAt time.Time `db:"created_at"`
}
```

Notes:
- `Role` is one of `"user"`, `"assistant"`, `"system"`
- `UserID` is only set for `role="user"` — identifies which person posted
- `ItemID` is nullable — most messages are general cart notes, some reference an item
- The `Text` field stores plain text. The frontend handles rendering `@ItemName`
  chips based on `ItemID`
- No editing or deleting messages for now — append-only log

## Schema

```sql
CREATE TABLE IF NOT EXISTS messages(
    id text PRIMARY KEY,
    cart_id text NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    role text NOT NULL DEFAULT 'user',
    user_id text REFERENCES users(user_id) ON DELETE SET NULL,
    text text NOT NULL,
    item_id text REFERENCES items(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Key decisions:
- `ON DELETE CASCADE` on cart — delete cart = delete all messages
- `ON DELETE SET NULL` on item — deleting an item keeps the message but clears the link
- `ON DELETE SET NULL` on user — deleting a user keeps the message but clears the author
- `role` is a plain text column, validated in Go code, not at the DB level

## Repository Methods

```go
// Add a message to a cart
func (r *SqliteRepository) AddMessage(msg *Message) error

// Load messages for a cart, ordered by created_at ASC
func (r *SqliteRepository) Messages(cartID string) ([]*Message, error)
```

`AddMessage` is a simple INSERT. `Messages` is a simple SELECT + ORDER BY.

We do NOT eagerly load messages when loading a cart — they're fetched separately
when the chat panel is rendered.

## Item Linking

When a message references an item:
- The message stores `item_id` (the item's ID in our DB)
- The frontend renders it as a chip showing the item's text/name
- If the item gets deleted, `item_id` becomes NULL (SET NULL), and the chip
  disappears — the message text still reads fine on its own

The item link is set at message creation time. The caller provides the item ID
if the message is about a specific item.

## Roles

- **user**: A person posted via the chat input. `UserID` is set.
- **assistant**: The LLM agent responded. `UserID` is nil.
- **system**: Automated events (e.g. "item added", "sold out"). `UserID` is nil.

The frontend styles each role differently (e.g. left/right bubbles, system
events as centered muted text).

## Future: Quick Actions (not implemented now)

Events like "did not find" or "sold out" will be messages with `role="system"`,
a fixed `Text`, and a linked `ItemID`. They'll be created by tapping a
quick-action button on an item while shopping. This is just regular
`AddMessage` calls with predefined text — no special schema needed.

## Tests

```
TestAddMessage
  - create cart, add user message without item link, fetch back, verify fields + role

TestAddMessageWithItemLink
  - create cart + item, add message linking the item, fetch back, verify item_id

TestMessageSurvivesItemDelete
  - create cart + item + linked message
  - delete the item
  - fetch message, verify it still exists but item_id is NULL

TestMessagesOrderedByTime
  - add 3 messages with different roles (user, assistant, system)
  - verify they come back in chronological order

TestMessageRoles
  - add one message per role, fetch back, verify role and user_id (set/nil)
```
