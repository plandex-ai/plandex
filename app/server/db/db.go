package db

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var Conn *sqlx.DB

func Connect() error {
	var err error

	if os.Getenv("DATABASE_URL") == "" {
		return errors.New("DATABASE_URL not set")
	}

	Conn, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	return nil
}

func MigrationsUp() error {
	if Conn == nil {
		return errors.New("db not initialized")
	}

	driver, err := postgres.WithInstance(Conn.DB, &postgres.Config{})

	if err != nil {
		return fmt.Errorf("error creating postgres driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)

	if err != nil {
		return fmt.Errorf("error creating migration instance: %v", err)
	}

	// Uncomment below (and update migration version) to reset migration state to a specific version after a failure
	// if os.Getenv("GOENV") == "development" {
	// migrateVersion := 2024011700
	//	 if err := m.Force(migrateVersion); err != nil {
	//	 	return fmt.Errorf("error forcing migration version: %v", err)
	// 	}
	// }

	// Uncomment below to run down migrations (RESETS DATABASE!!)
	if os.Getenv("GOENV") == "development" {
		err = m.Down()
		if err != nil {
			if err == migrate.ErrNoChange {
				log.Println("no migrations to run down")
			} else {
				return fmt.Errorf("error running down migrations: %v", err)
			}
		}
		log.Println("ran down migrations - database was reset")
	}

	err = m.Up()

	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("migration state is up to date")
		} else {

			return fmt.Errorf("error running migrations: %v", err)
		}
	}

	if err == nil {
		log.Println("ran migrations successfully")
	}

	return nil
}
