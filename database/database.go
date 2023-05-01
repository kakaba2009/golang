package database

import (
	"database/sql"
	"log"
)

var db *sql.DB

func init() {
	var err error

	db, err = sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		log.Fatal(err)
	}
}

func DB() *sql.DB {
	return db
}
