package db

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigration(migrationURL string, dbURL string) {
	m, err := migrate.New(migrationURL, dbURL)
	if err != nil {
		log.Fatalf("Init migration failed: %v", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database is up to date.")
		} else {
			log.Fatalf("Migration failed: %v", err)
		}
	} else {
		log.Println("Migration successful!")
	}
}
