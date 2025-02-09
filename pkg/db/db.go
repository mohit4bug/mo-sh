package db

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var (
	db   *sql.DB
	once sync.Once
)

func InitDB() {
	once.Do(func() {
		connStr := "user=user password=password dbname=db sslmode=disable"
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Fatal(err)
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(time.Duration(30) * time.Minute)
		db.SetConnMaxIdleTime(time.Duration(5) * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Fatal(err)
		}
	})
}

func GetDB() *sql.DB {
	if db == nil {
		log.Panic("DB is not initialized")
	}
	return db
}

func CloseDB() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		} else {
			log.Println("DB connection closed")
		}
	}
}
