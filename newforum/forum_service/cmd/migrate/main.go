package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var (
		forceVersion int
		dropDb       bool
	)
	flag.IntVar(&forceVersion, "force-version", -1, "Force database version")
	flag.BoolVar(&dropDb, "drop", false, "Drop all tables before running migrations")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Stas2005101010!@localhost:5432/forum?sslmode=disable"
	}

	log.Printf("Connecting to database: %s", dbURL)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	if dropDb {
		log.Println("Dropping all tables...")
		if _, err := db.Exec(`
			DROP TABLE IF EXISTS schema_migrations CASCADE;
			DROP TABLE IF EXISTS global_messages CASCADE;
			DROP TABLE IF EXISTS messages CASCADE;
			DROP TABLE IF EXISTS forums CASCADE;
		`); err != nil {
			log.Fatalf("Error dropping tables: %v", err)
		}
		log.Println("All tables dropped successfully")
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Error initializing driver: %v", err)
	}

	// Get absolute path to migrations
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		log.Fatalf("Error getting migrations path: %v", err)
	}
	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath))
	log.Printf("Using migrations from: %s", sourceURL)

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Error creating migration instance: %v", err)
	}

	if forceVersion >= 0 {
		log.Printf("Forcing database version to %d", forceVersion)
		if err := m.Force(forceVersion); err != nil {
			log.Fatalf("Error forcing version: %v", err)
		}
		log.Printf("Successfully forced version to %d", forceVersion)
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error running migrations: %v", err)
	}
	log.Println("Migrations applied successfully")
}
