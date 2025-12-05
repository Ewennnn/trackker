package service

import (
	"bufio"
	"djtracker/internal/config"
	"djtracker/internal/model"
	"djtracker/internal/repository"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Service struct {
	log           *slog.Logger
	config        *config.Config
	repo          *repository.Repository
	reader        *bufio.Reader
	liveTracklist chan string

	trackBroadcaster *Broadcaster[*model.TrackDTO]
}

func New(log *slog.Logger, config *config.Config, repo *repository.Repository) *Service {
	return &Service{
		log:           log,
		config:        config,
		repo:          repo,
		liveTracklist: make(chan string, 1),

		trackBroadcaster: NewBroadcaster[*model.TrackDTO](log),
	}
}

// SubscribeForTracks Créer un nouveau channel abonné à la réception des tracks
func (s *Service) SubscribeForTracks() (chan *model.TrackDTO, func()) {
	return s.trackBroadcaster.Subscribe(1)
}

func (s *Service) StartTracking() error {
	file, err := os.Open(s.config.Tracker.Path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	_, err = file.Seek(stat.Size(), 0)
	if err != nil {
		return err
	}

	go s.readTracks(file)
	go s.analyseTracks()
	return nil
}

// analyseTracks Lit les tracks brutes reçues de liveTracklist
// Transfer les informations de la TrackDTO vers le channel Tracks
func (s *Service) analyseTracks() {
	for trackText := range s.liveTracklist {
		track := &model.TrackDTO{
			Name: trackText,
		}

		s.repo.AddTrackToHistory(track)
		s.trackBroadcaster.Broadcast(track)
	}
}

// readTracks Lit le fichier tracklist de VirtualDJ
// Transfer les informations brutes vers le channel liveTracklist
func (s *Service) readTracks(file *os.File) {
	reader := bufio.NewReader(file)
	defer s.handleClose(file)
	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			s.log.Error("Error while reading file", err)
		}
		data = strings.TrimRight(data, "\r\n")

		s.liveTracklist <- data
	}
}

func (s *Service) handleClose(file *os.File) {
	err := file.Close()
	if err != nil {
		s.log.Error("Failed to close file", err)
	}
}
