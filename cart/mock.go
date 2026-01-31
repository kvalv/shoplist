package cart

import (
	"fmt"
	"log"
	"time"
)

func NewMock() Repository {
	repo, err := NewSqlite(":memory:")
	if err != nil {
		panic(err)
	}
	return repo
}

func SetupMockData(repo Repository) {
	c, _ := repo.New()
	_, week := time.Now().ISOWeek()
	c.Name = fmt.Sprintf("Uke %d", week)
	c.Add("bringebærsyltetøy").SetChecked()
	c.Add("brus til meg")
	c.Add("saft")
	if err := repo.Save(c); err != nil {
		log.Fatalf("failed to save: %s", err)
	}
}
