package main

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:generate sqlc generate

//go:embed sql/migrations/*.sql
var migrations embed.FS

// opens the database connection and creates the schema
func create_db(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return db, db.Ping()
}

// TODO: create backup before migration
func migrate_db(ctx context.Context, db *sql.DB) error {
	// create driver and source
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}
	source, err := iofs.New(migrations, "sql/migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// create migration instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	// execute migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate db to new version: %w", err)
	}
	v, d, err := m.Version()
	slog.Info("Migrated db", "version", v, "dirty", d, "err", err)
	return nil
}
