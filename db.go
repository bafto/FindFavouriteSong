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
	sqlite3_driver "github.com/mattn/go-sqlite3"
)

//go:generate sqlc generate

//go:embed sql/migrations/*.sql
var migrations embed.FS

// opens the database connection
func create_db(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return db, db.Ping()
}

func migrate_db(ctx context.Context, db *sql.DB) error {
	slog.Info("creating database backup before migration", "backup-path", config.BackupPath)
	// backupDest
	backupDest, err := create_db(ctx, config.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup db: %w", err)
	}
	defer backupDest.Close()

	if err := backup_db(ctx, backupDest, db); err != nil {
		return fmt.Errorf("failed to backup db: %w", err)
	}
	slog.Info("done creating database backup")

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

func backup_db(ctx context.Context, dest, src *sql.DB) error {
	destConn, err := dest.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish connection to backup destination db: %w", err)
	}
	defer destConn.Close()

	srcConn, err := src.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish connection to backup source db: %w", err)
	}
	defer srcConn.Close()

	return destConn.Raw(func(destConn any) error {
		return srcConn.Raw(func(srcConn any) error {
			destSQLiteConn, ok := destConn.(*sqlite3_driver.SQLiteConn)
			if !ok {
				return fmt.Errorf("can't convert destination connection to SQLiteConn")
			}

			srcSQLiteConn, ok := srcConn.(*sqlite3_driver.SQLiteConn)
			if !ok {
				return fmt.Errorf("can't convert source connection to SQLiteConn")
			}

			b, err := destSQLiteConn.Backup("main", srcSQLiteConn, "main")
			if err != nil {
				return fmt.Errorf("error initializing SQLite backup: %w", err)
			}

			done, err := b.Step(-1)
			if !done {
				return fmt.Errorf("step of -1, but not done")
			}
			if err != nil {
				return fmt.Errorf("error in stepping backup: %w", err)
			}

			err = b.Finish()
			if err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return err
		})
	})
}
