package model

import (
	"time"
)

type Track struct {
	ID       int64          `db:"id"`
	EventID  int64          `db:"event_id"`
	Artist   *string        `db:"artist"`
	Name     string         `db:"name"`
	PlayAt   time.Time      `db:"play_at"`
	Path     *string        `db:"path"`
	Duration *time.Duration `db:"duration"`
	Cover    *string        `db:"cover"`
}

func (t *Track) IsFinished(now time.Time) bool {
	if t.Duration == nil {
		return true
	}
	end := t.PlayAt.Add(*t.Duration)
	return end.Before(now)
}

func (t *Track) ToDTO() *TrackDTO {
	return &TrackDTO{
		Artist:   t.Artist,
		Name:     t.Name,
		PlayAt:   t.PlayAt.Format(time.RFC3339),
		Duration: t.Duration,
		Cover:    t.Cover,
	}
}

type TrackDTO struct {
	Artist   *string        `json:"artist,omitempty"`
	Name     string         `json:"name"`
	PlayAt   string         `json:"play_at"`
	Duration *time.Duration `json:"duration,omitempty"`
	Cover    *string        `json:"cover,omitempty"`
}
