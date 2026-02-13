package carts

import gonanoid "github.com/matoous/go-nanoid/v2"

func newID() string {
	return gonanoid.MustGenerate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 8)
}
