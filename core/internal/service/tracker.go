package service

import (
	"bufio"
	"context"
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

func NewTracker(log *slog.Logger, config *config.Config, repo *repository.Repository, parser parser.Parser) *Tracker {
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

func (t *Tracker) StartTracking(ctx context.Context) {
	go t.superviseHistoryReader(ctx)
	go t.listenHistory(ctx)
}

func (t *Tracker) superviseHistoryReader(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := t.parser.WithHistoryTrackReader(func(reader *bufio.Reader) error {
				t.log.Info("Ready to read tracks history")
				return t.handleHistoryReader(ctx, reader)
			})

			if err != nil {
				t.log.Error("history reader crashed", "err", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
			}
		}
	}
}

func (t *Tracker) handleHistoryReader(ctx context.Context, reader *bufio.Reader) error {
	lastSavedTrack, _ := t.repo.FindLastTrack()

	tracks, err := t.parser.UpdateHistory(reader, lastSavedTrack)
	if err != nil {
		t.log.Error("Failed to update history", "err", err)
	}

	if len(tracks) > 0 {
		t.processTracks(tracks)
	}

	return t.parser.StartHistoryTracking(ctx, reader, t.liveTrackList)
}

func (t *Tracker) processTracks(tracks []*model.Track) {
	t.saveHistoryTracks(tracks[:len(tracks)-1])
	t.handleLastTrack(tracks[len(tracks)-1])
}

func (t *Tracker) saveHistoryTracks(tracks []*model.Track) {
	for _, track := range tracks {
		t.repo.AddTrackToHistory(track)
	}
}

func (t *Tracker) handleLastTrack(track *model.Track) {
	if track.IsFinished(time.Now()) {
		t.repo.AddTrackToHistory(track)
		return
	}

	t.liveTrackList <- track
}

// listenHistory Reçoit les Tracks traités par le Parser
// et les envoie dans les différents canaux de diffusion
func (t *Tracker) listenHistory(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case track := <-t.liveTrackList:
			t.repo.AddTrackToHistory(track)
			t.trackBroadcaster.Broadcast(track)
		}
	}
}
