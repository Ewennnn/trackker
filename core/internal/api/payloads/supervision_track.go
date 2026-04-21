package payloads

import (
	"net/http"
	"trackker/internal/model"
	"trackker/internal/utils"
)

type SupervisionTrackPayload struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	FilePath string  `json:"filePath"`
	CoverURL *string `json:"coverUrl"`
}

func BuildSupervisionTrackPayload(r *http.Request, track *model.Track) *SupervisionTrackPayload {
	if track == nil {
		return nil
	}

	artist := ""
	if track.Artist != nil {
		artist = *track.Artist
	}

	coverURL := utils.GetCoverURL(r, track.ID)
	return &SupervisionTrackPayload{
		Title:    track.Name,
		Artist:   artist,
		FilePath: track.Path,
		CoverURL: &coverURL,
	}
}
