package data

import (
	"database/sql"
	"fmt"

	"github.com/barelyhuman/go/env"
)

func GetDatabaseHandle() (*sql.DB, error) {
	dsn := env.Get("DATABASE_URL", "")
	if len(dsn) == 0 {
		return &sql.DB{}, fmt.Errorf("failed to get DSN for database, DATABASE_URL is a required env variable")
	}
	db, err := sql.Open("sqlite3", "./data.sqlite3")
	return db, err
}
