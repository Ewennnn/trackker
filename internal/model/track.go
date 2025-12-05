package model

import "time"

type Track struct {
	ID       int64     `db:"id"`
	EventID  int64     `db:"event_id"`
	Artist   string    `db:"artist"`
	Name     string    `db:"name"`
	PlayTime time.Time `db:"play_at"`
}

type TrackDTO struct {
	Artist string
	Name   string
	PlayAt string
}
