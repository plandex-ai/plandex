package db

import (
	"errors"
	"fmt"
	"log"
	"net/url"
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

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		if os.Getenv("DB_HOST") != "" &&
			os.Getenv("DB_PORT") != "" &&
			os.Getenv("DB_USER") != "" &&
			os.Getenv("DB_PASSWORD") != "" &&
			os.Getenv("DB_NAME") != "" {
			encodedPassword := url.QueryEscape(os.Getenv("DB_PASSWORD"))

			dbUrl = "postgres://" + os.Getenv("DB_USER") + ":" + encodedPassword + "@" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + "/" + os.Getenv("DB_NAME")
		}

		if dbUrl == "" {
			return errors.New("DATABASE_URL or DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, and DB_NAME environment variables must be set")
		}
	}

	Conn, err = sqlx.Connect("postgres", dbUrl)
	if err != nil {
		return err
	}

	log.Println("connected to database")

	_, err = Conn.Exec("SET TIMEZONE='UTC';")

	if err != nil {
		return fmt.Errorf("error setting timezone: %v", err)
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
	// 	migrateVersion := 2024013000
	// 	if err := m.Force(migrateVersion); err != nil {
	// 		return fmt.Errorf("error forcing migration version: %v", err)
	// 	}
	// }

	// Uncomment below to run down migrations (RESETS DATABASE!!)
	// if os.Getenv("GOENV") == "development" {
	// 	err = m.Down()
	// 	if err != nil {
	// 		if err == migrate.ErrNoChange {
	// 			log.Println("no migrations to run down")
	// 		} else {
	// 			return fmt.Errorf("error running down migrations: %v", err)
	// 		}
	// 	}
	// 	log.Println("ran down migrations - database was reset")
	// }

	// Uncomment below to go back ONE migration
	// if os.Getenv("GOENV") == "development" {
	// 	err = m.Steps(-1)
	// 	if err != nil {
	// 		return fmt.Errorf("error running down migrations: %v", err)
	// 	}
	// 	log.Println("went down 1 migration")
	// }

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
