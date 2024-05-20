package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	spotify "github.com/zmb3/spotify/v2"

	_ "modernc.org/sqlite"
)

const (
	create_schema = `
CREATE TABLE IF NOT EXISTS playlist (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL
)
`
	table_exists_query = `
SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='playlist'
`

	test_insert = `
INSERT INTO playlist (name) VALUES ('test'), ('test2'), ('test3')
`
)

func create_db(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return nil, err
		}

		_, err = os.Create(path)
		if err != nil {
			return nil, err
		}
		slog.Println("Created database", "path", path)
	} else if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	return db, db.Ping()
}

func main() {
	db, err := create_db("fff.db")
	if err != nil {
		slog.Error("Error creating database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Connected to database")

	table_count := 0
	if err := db.QueryRow(table_exists_query).Scan(&table_count); err != nil {
		slog.Error("Unable to get table count", "error", err)
		os.Exit(1)
	}

	slog.Info("Got Table count", "table count", table_count)
	if table_count > 0 {
		result, err := db.Query("SELECT * FROM playlist")
		if err != nil {
		slog.Error("Unable to get table count", "error", err)
		os.Exit(1)
		}
		defer result.Close()

		for result.Next() {
			var id int
			var name string
			if err := result.Scan(&id, &name); err != nil {
				slog.Fatal(err)
			}
			slog.Printf("id: %d, name: %s\n", id, name)
		}
		return
	}

	_, err = db.Exec(create_schema)
	if err != nil {
		slog.Fatal(err)
		slog.Info
	}
	slog.Println("Created schema")

	_, err = db.Exec(test_insert)
	if err != nil {
		slog.Fatal(err)
	}
	slog.Println("Inserted test data")

	spClient := new(spotify.Client)
	page, _ := spClient.GetPlaylistItems(context.TODO(), "23")
	page.Items[0].Track.Track.
}


