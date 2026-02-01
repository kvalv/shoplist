package cart

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

func NewMock() Repository {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	repo, err := NewSqlite(db)
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
