package main

import (
	"log"

	"github.com/kvalv/shoplist/cart"
)

func SetupMockData(repo cart.Repository) {
	cart, _ := repo.New()
	cart.Add("bringebærsyltetøy").SetChecked()
	cart.Add("brus til meg")
	cart.Add("saft")
	if err := repo.Save(cart); err != nil {
		log.Fatalf("failed to save: %s", err)
	}
}
