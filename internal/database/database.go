package database

import (
	"database/sql"
	"djtracker/internal/config"
	"djtracker/internal/utils"
	"os"
	"path/filepath"
)

func createDbPath(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	return nil
}

func UseDb(conf *config.Config, app func(db *sql.DB) error) error {
	if err := createDbPath(conf.Database.Path); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", conf.Database.Path)
	if err != nil {
		return err
	}
	defer utils.SafeClose(db)

	if err := db.Ping(); err != nil {
		return err
	}

	if err := migrate(db); err != nil {
		return err
	}

	return app(db)
}

func migrate(db *sql.DB) error {
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
			duration INTEGER,
			path TEXT,
			
			FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE 
		)
	`)
	return err
}
