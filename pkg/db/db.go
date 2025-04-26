package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewDatabase(databaseURL string) *sqlx.DB {
	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	return db
}
