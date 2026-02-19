package db

import (
	"errors"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) {
	if databaseURL == "" {
		log.Println("No database URL provided, skipping migrations")
		return
	}

	var m *migrate.Migrate
	var err error

	for i := 0; i < 10; i++ {
		m, err = migrate.New(
			"file://migrations",
			databaseURL,
		)
		if err == nil {
			break
		}
		log.Printf("Migration setup failed, retrying in 2s: %v", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Migration setup failed after retries: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations completed successfully")
}
