package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Bethel-nz/tickit/internal/env"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	flag.Parse()
	var dbURL = env.String("DATABASE_URL", "", env.Require).Get()
	var migrationsPath = env.String("MIGRATIONS_PATH", "internal/database/migrations", env.Optional).Get()

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	log.Println("Migrations applied successfully!")
}
