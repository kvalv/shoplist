package carts

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kvalv/shoplist/migrations"
	_ "modernc.org/sqlite"
)

func NewMock() (*SqliteRepository, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	if err := migrations.Migrate(db); err != nil {
		panic(err)
	}
	repo, err := NewRepository(db)
	if err != nil {
		panic(err)
	}
	return repo, db
}

func SetupMockData(repo *SqliteRepository) {
	// c, _ := repo.New()
	c := New()
	_, week := time.Now().ISOWeek()
	// suffix := gonanoid.Must(3)
	c.Name = fmt.Sprintf("Uke %d %s", week, c.ID)
	c.Add("bringebærsyltetøy").SetChecked()
	c.Add("brus til meg")
	c.Add("saft")
	if err := repo.Save(c); err != nil {
		log.Fatalf("failed to save: %s", err)
	}
}
