package model

import (
	"time"
)

type TrackModel struct {
	Artist   *string
	Name     string
	PlayAt   time.Time
	Duration *time.Duration
	Cover    *string
}

type Track struct {
	ID      int64     `db:"id"`
	EventID int64     `db:"event_id"`
	Artist  *string   `db:"artist"`
	Name    string    `db:"name"`
	PlayAt  time.Time `db:"play_at"`
}

func (t *TrackModel) ToEntity() *Track {
	return &Track{
		Artist: t.Artist,
		Name:   t.Name,
		PlayAt: t.PlayAt,
	}
}

type TrackDTO struct {
	Artist   *string        `json:"artist,omitempty"`
	Name     string         `json:"name"`
	PlayAt   string         `json:"play_at"`
	Duration *time.Duration `json:"duration,omitempty"`
	Cover    *string        `json:"cover,omitempty"`
}

func (t *TrackModel) ToDTO() *TrackDTO {
	return &TrackDTO{
		Artist:   t.Artist,
		Name:     t.Name,
		PlayAt:   t.PlayAt.Format(time.RFC3339),
		Duration: t.Duration,
		Cover:    t.Cover,
	}
}
