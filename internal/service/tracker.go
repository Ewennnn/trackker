package service

import (
	"bufio"
	"djtracker/internal/config"
	"djtracker/internal/model"
	"djtracker/internal/repository"
	"djtracker/internal/utils"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Tracker struct {
	log           *slog.Logger
	config        *config.Config
	repo          *repository.Repository
	reader        *bufio.Reader
	liveTracklist chan string

	trackBroadcaster *Broadcaster[*model.Track]
}

func New(log *slog.Logger, config *config.Config, repo *repository.Repository) *Tracker {
	return &Tracker{
		log:           log,
		config:        config,
		repo:          repo,
		liveTracklist: make(chan string, 1),

		trackBroadcaster: NewBroadcaster[*model.Track](log),
	}
}

// SubscribeForTracks Créer un nouveau channel abonné à la réception des tracks
func (t *Tracker) SubscribeForTracks() (chan *model.Track, func()) {
	return t.trackBroadcaster.Subscribe(1)
}

// GetCurrentTrack Récupère la track actuelle et l'envoie dans le channel
func (t *Tracker) GetCurrentTrack() *model.Track {
	track, err := t.repo.FindLastTrack()
	if err != nil {
		t.log.Error("Failed to retrieve current track", err)
		return nil
	}

	if track == nil {
		t.log.Debug("No current track was found")
		return nil
	}

	if track.IsFinished(time.Now()) {
		t.log.Debug("Last track finished")
		return nil
	}

	return track
}

func (t *Tracker) StartTracking() error {
	tracklistFile, err := os.Open(t.config.Tracker.History.Path)
	if err != nil {
		return err
	}

	stat, err := tracklistFile.Stat()
	if err != nil {
		return err
	}
	_, err = tracklistFile.Seek(stat.Size(), 0)
	if err != nil {
		return err
	}

	go t.readTracks(tracklistFile)
	go t.analyseTracks()
	return nil
}

// analyseTracks Lit les tracks brutes reçues de liveTracklist
// Transfer les informations de la TrackDTO vers le channel Tracks
func (t *Tracker) analyseTracks() {
	for trackText := range t.liveTracklist {
		parsedTrack, err := ParseLine(trackText)
		if err != nil {
			t.log.Error("Unable to parse track line", "track_line", trackText)
			continue
		}

		track := &model.Track{
			Artist: utils.SafeTrim(parsedTrack.Artist),
			Name:   strings.TrimSpace(parsedTrack.Name),
			PlayAt: time.Now(),
		}

		// Récupération des données du fichier de la track
		fileTrackData, err := t.findTrackFile(track.Name)
		if err != nil {
			t.log.Error("Track file not found", "track", track.Name)
			t.repo.AddTrackToHistory(track)
			t.trackBroadcaster.Broadcast(track)
			continue
		}
		t.log.Debug("Track file found", "track", fileTrackData)
		track.Path = &fileTrackData.Path

		// Ouverture du fichier de la track
		trackFile, err := os.Open(fileTrackData.Path)
		if err != nil {
			t.log.Error("Failed to open track file", "path", fileTrackData.Path)
			t.repo.AddTrackToHistory(track)
			t.trackBroadcaster.Broadcast(track)
			continue
		}

		// Récupération de la durée de la track
		if duration, err := t.findTrackDuration(trackFile, fileTrackData.MapExtType()); err == nil {
			track.Duration = &duration
		} else {
			t.log.Error("Failed to retrieve track duration", "track", fileTrackData.Name, "path", fileTrackData.Path)
		}

		// Récupération de la cover de la track
		if cover, err := t.findTrackCover(trackFile); err == nil {
			track.Cover = &cover
		} else {
			t.log.Error("Failed to retrieve track cover", "track", fileTrackData.Name, "path", fileTrackData.Path)
		}

		utils.SafeClose(trackFile)
		t.repo.AddTrackToHistory(track)
		t.trackBroadcaster.Broadcast(track)
	}
}

func (t *Tracker) findTrackFile(track string) (*FileTrackData, error) {
	for _, sourceFolder := range t.config.Tracker.Source.Paths {
		if file, err := FindFile(sourceFolder, track); err == nil {
			return file, nil
		}
	}
	return nil, fmt.Errorf("track file not found for %s", track)
}

// readTracks Lit le fichier tracklist de VirtualDJ
// Transfer les informations brutes vers le channel liveTracklist
func (t *Tracker) readTracks(file *os.File) {
	reader := bufio.NewReader(file)
	defer utils.SafeClose(file)
	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			t.log.Error("Error while reading file", err)
		}
		data = strings.TrimRight(data, "\r\n")

		t.liveTracklist <- data
	}
}
