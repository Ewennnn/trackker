package formatter

import (
	"djtracker/internal/model"
	"encoding/json"
	"time"
)

type trackDTO struct {
	ID       int64          `json:"id"`
	Artist   *string        `json:"artist,omitempty"`
	Name     string         `json:"name"`
	PlayAt   string         `json:"play_at"`
	Duration *time.Duration `json:"duration,omitempty"`
	Cover    *string        `json:"cover,omitempty"`
}

func newTrackDTO(t *model.Track) *trackDTO {
	return &trackDTO{
		ID:       t.ID,
		Artist:   t.Artist,
		Name:     t.Name,
		PlayAt:   t.PlayAt.Format(time.RFC3339),
		Duration: t.Duration,
		Cover:    t.Cover,
	}
}

type JsonFormatter struct{}

func (p *JsonFormatter) Format(track *model.Track) (string, error) {
	dto := newTrackDTO(track)
	data, err := json.Marshal(dto)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
