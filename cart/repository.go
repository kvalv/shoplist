package cart

type Repository interface {
	New() (*Cart, error)
	Save(cart *Cart) error
	Update(item *Item) error
	Latest() (*Cart, error)
	Cart(ID string) (*Cart, error)
}
