package migrations

import (
	"database/sql"
	_ "embed"
)

//go:embed migration.sql
var migrationSQL string

func Migrate(db *sql.DB) error {
	_, err := db.Exec(migrationSQL)
	return err
}
