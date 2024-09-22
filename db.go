package main

import (
	"context"
	"database/sql"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:generate sqlc generate

//go:embed sql/schema.sql
var db_schema string

// opens the database connection and creates the schema
func create_db(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// create tables
	if _, err := db.ExecContext(ctx, db_schema); err != nil {
		return nil, err
	}

	return db, db.Ping()
}
