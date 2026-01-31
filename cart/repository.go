package cart

type Repository interface {
	New() (*Cart, error)
	Add(CartID string, item Item) error
	Save(cart *Cart) error
	Latest() (*Cart, error)
}
