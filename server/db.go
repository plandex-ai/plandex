package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
)

var db *sqlx.DB

func InitDB(dataSourceName string) error {
	var err error
	db, err = sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		return err
	}
	return nil
}
