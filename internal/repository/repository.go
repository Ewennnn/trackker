package repository

import (
	"database/sql"
	"djtracker/internal/model"
	"errors"
	"fmt"
	"log/slog"
	_ "modernc.org/sqlite"
	"time"
)

type Repository struct {
	log   *slog.Logger
	db    *sql.DB
	event *model.Event
}

func New(log *slog.Logger, db *sql.DB) *Repository {
	return &Repository{
		log: log,
		db:  db,
	}
}

func (r *Repository) PrepareEvent() error {
	var last model.Event
	err := r.db.QueryRow(`
		SELECT id, start
		FROM events
		ORDER BY start DESC
		LIMIT 1
		`).Scan(&last.ID, &last.Start)

	now := time.Now()

	if errors.Is(err, sql.ErrNoRows) {
		return r.createNewEvent(now)
	}

	if err != nil {
		return fmt.Errorf("error fetching last event: %w", err)
	}

	lastDate := last.Start.Truncate(24 * time.Hour)
	today := now.Truncate(24 * time.Hour)

	isNextDay := today.After(lastDate)
	isAfterNine := now.Hour() > 9

	if isNextDay && isAfterNine {
		return r.createNewEvent(now)
	}

	r.log.Info("Load current event", "event", fmt.Sprintf("%#v", last))
	r.event = &last
	return nil
}

func (r *Repository) createNewEvent(date time.Time) error {
	res, err := r.db.Exec(`
		INSERT INTO events (start) VALUES (?)
	`, date)
	if err != nil {
		return fmt.Errorf("error creating new event: %w", err)
	}

	id, _ := res.LastInsertId()
	r.event = &model.Event{
		ID:    id,
		Start: date,
	}
	r.log.Info("New event created", "event", fmt.Sprintf("%#v", r.event))
	return nil
}

func (r *Repository) AddTrackToHistory(track *model.Track) {
	_, err := r.db.Exec(`
		INSERT INTO tracks (event_id, artist, name, play_at) VALUES (?, ?, ?, ?)
	`, r.event.ID, track.Artist, track.Name, track.PlayAt)

	if err != nil {
		r.log.Warn("Failed to insert track into history", "event", r.event.ID, "track", fmt.Sprintf("%#v", track))
	} else {
		r.log.Info("Track successfully saved", "event", r.event.ID, "track", fmt.Sprintf("%#v", track))
	}
}
