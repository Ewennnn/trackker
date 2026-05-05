package model

import (
	"net/http"
	"time"
	"trackker/internal/utils"
)

type Track struct {
	ID       int64         `db:"id"`
	EventID  int64         `db:"event_id"`
	Artist   *string       `db:"artist"`
	Name     string        `db:"name"`
	PlayAt   time.Time     `db:"play_at"`
	Path     string        `db:"path"`
	Duration time.Duration `db:"duration"`
}

func (t *Track) IsFinished(now time.Time) bool {
	end := t.PlayAt.Add(t.Duration)
	return end.Before(now)
}

// Equals compare deux tracks pour vérifier si elles sont identiques
func (t *Track) Equals(other *Track) bool {
	if t == nil || other == nil {
		return false
	}
	return utils.StringPtrEqual(t.Artist, other.Artist) &&
		t.Name == other.Name &&
		t.Path == other.Path &&
		t.PlayAt == other.PlayAt
}

type SimpleTrackResponse struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	FilePath string  `json:"filePath"`
	CoverURL *string `json:"coverUrl"`
}

func BuildSupervisionTrackPayload(r *http.Request, track *Track) *SimpleTrackResponse {
	if track == nil {
		return nil
	}

	artist := ""
	if track.Artist != nil {
		artist = *track.Artist
	}

	coverURL := utils.GetCoverURL(r, track.ID)
	return &SimpleTrackResponse{
		Title:    track.Name,
		Artist:   artist,
		FilePath: track.Path,
		CoverURL: &coverURL,
	}
}
