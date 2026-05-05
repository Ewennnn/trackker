package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"trackker/internal/config"
	"trackker/internal/utils"
)

func createDbPath(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	return nil
}

func Connect(conf *config.Config) (*sql.DB, error) {
	if err := createDbPath(conf.Database.Path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", conf.Database.Path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		utils.SafeClose(db)
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	if err := createEventsTable(db); err != nil {
		return err
	}

	if err := createTracksTable(db); err != nil {
		return err
	}

	if err := createButtonsTable(db); err != nil {
		return err
	}

	if err := populateSystemButtons(db); err != nil {
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

func createButtonsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS buttons (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    label varchar(255) NOT NULL,
		    text_color varchar(7) NOT NULL,
		    background_color varchar(7)	NOT NULL,
		    icon varchar(255),
		    display_mode varchar(255) NOT NULL,
		    position INTEGER UNIQUE NOT NULL,
		    is_deletable boolean NOT NULL
		)
	`)
	return err
}

func populateSystemButtons(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO buttons (id, label, text_color, background_color, display_mode, position, is_deletable)
		VALUES 
			(1, 'Blackout', '#FFFFFF', '#b110e7', 'blackout', 1, false),
			(2, 'Freeze Tracking', '#FFFFFF', '#177fea', 'freeze_tracking', 2, false)
		ON CONFLICT DO NOTHING;
	`)

	return err
}
