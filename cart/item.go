package cart

// Represents a single item to buy
type Item struct {
	ID      string
	Text    string
	Checked bool
}

func (i *Item) SetChecked(value ...bool) {
	if len(value) > 0 {
		i.Checked = value[0]
		return
	}
	i.Checked = true
}
func (i *Item) Toggle() {
	i.Checked = !i.Checked
}
