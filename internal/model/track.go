package model

import (
	"time"
)

type Track struct {
	ID      int64     `db:"id"`
	EventID int64     `db:"event_id"`
	Artist  *string   `db:"artist"`
	Name    string    `db:"name"`
	PlayAt  time.Time `db:"play_at"`
}

type TrackDTO struct {
	Artist *string `json:"artist,omitempty"`
	Name   string  `json:"name"`
	PlayAt string  `json:"play_at"`
}

func (t *Track) ToDTO() *TrackDTO {
	return &TrackDTO{
		Artist: t.Artist,
		Name:   t.Name,
		PlayAt: t.PlayAt.Format(time.RFC3339),
	}
}
