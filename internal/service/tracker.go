package service

import (
	"bufio"
	"djtracker/internal/config"
	"djtracker/internal/model"
	"djtracker/internal/repository"
	"djtracker/internal/service/parser"
	"log/slog"
	"time"
)

type Tracker struct {
	log    *slog.Logger
	config *config.Config
	repo   *repository.Repository

	parser        parser.Parser
	liveTrackList chan *model.Track

	trackBroadcaster *Broadcaster[*model.Track]
}

func New(log *slog.Logger, config *config.Config, repo *repository.Repository, parser parser.Parser) *Tracker {
	return &Tracker{
		log:    log,
		config: config,
		repo:   repo,

		parser:        parser,
		liveTrackList: make(chan *model.Track, 1),

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

func (t *Tracker) StartTracking() {
	go t.superviseHistoryReader()
	go t.listenHistory()
}

func (t *Tracker) superviseHistoryReader() {
	for {
		err := t.parser.WithHistoryTrackReader(func(reader *bufio.Reader) error {
			t.log.Info("Ready to read tracks history")
			return t.parser.StartHistoryTracking(reader, t.liveTrackList)
		})

		if err != nil {
			t.log.Error("history reader crashed", "err", err)
			time.Sleep(2 * time.Second)
		}
	}
}

// listenHistory Reçoit les Tracks traités par le Parser
// et les envoie dans les différents canaux de diffusion
func (t *Tracker) listenHistory() {
	for track := range t.liveTrackList {
		t.repo.AddTrackToHistory(track)
		t.trackBroadcaster.Broadcast(track)
	}
}
