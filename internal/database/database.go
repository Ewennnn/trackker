package database

import (
	"database/sql"
	"djtracker/internal/config"
	"log/slog"
	"os"
	"path/filepath"
)

func Init(config *config.Config) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(config.Database.Path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", config.Database.Path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB, log *slog.Logger) error {
	log.Debug("Initializing database")
	if err := createEventsTable(db); err != nil {
		return err
	}

	if err := createTracksTable(db); err != nil {
		return err
	}

	return nil
}

func createEventsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			start DATETIME NOT NULL 
		)
	`)

	return err
}

func createTracksTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			artist VARCHAR(255),
			name VARCHAR(255) NOT NULL,
			play_at DATETIME NOT NULL,
			
			FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE 
		)
	`)
	return err
}
