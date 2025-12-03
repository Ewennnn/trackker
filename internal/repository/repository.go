package repository

import (
	"djtracker/internal/model"
	"fmt"
	"log/slog"
)

type Repository struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Repository {
	return &Repository{
		log: log,
	}
}

func (r *Repository) AddTrackToHistory(track *model.Track) {
	fmt.Printf("New track added into history%#v+\n", track)
}
