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
	"sync"
)

type Service struct {
	log           *slog.Logger
	config        *config.Config
	repo          *repository.Repository
	reader        *bufio.Reader
	liveTracklist chan string

	mu      sync.RWMutex
	clients map[int]chan *model.Track
	nextId  int
}

func New(log *slog.Logger, config *config.Config, repo *repository.Repository) *Service {
	return &Service{
		log:           log,
		config:        config,
		repo:          repo,
		liveTracklist: make(chan string, 1),

		clients: make(map[int]chan *model.Track),
	}
}

// SubscribeForTracks Créer un nouveau channel abonné à la réception des tracks
func (s *Service) SubscribeForTracks() (chan *model.Track, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel := make(chan *model.Track, 1)

	id := s.nextId
	s.nextId++
	s.clients[id] = channel

	s.log.Info("Client just subscribed", "id", id)
	return channel, func() {
		s.unsubscribeForTracks(id)
	}
}

// unsubscribeForTracks Désabonne et supprime un channel abonné à la réception des tracks
func (s *Service) unsubscribeForTracks(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.clients[id]; ok {
		close(ch)
		delete(s.clients, id)
	}
	s.log.Info("Client just unsubscribe", "id", id)
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
// Transfer les informations de la Track vers le channel Tracks
func (s *Service) analyseTracks() {
	for trackText := range s.liveTracklist {
		track := &model.Track{
			Name: trackText,
		}

		s.repo.AddTrackToHistory(track)
		s.broadcastTrack(track)
	}
}

// broadcastTrack Diffuse à tous les clients abonnés lorsqu'une nouvelle track est diffusée
func (s *Service) broadcastTrack(track *model.Track) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.log.Info("Receive track to broadcast to clients: " + track.Name)
	for _, ch := range s.clients {
		ch <- track
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
