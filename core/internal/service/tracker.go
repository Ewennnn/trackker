package service

import (
	"bufio"
	"context"
	"log/slog"
	"time"
	"trackker/internal/config"
	"trackker/internal/model"
	"trackker/internal/repository"
	"trackker/internal/service/parser"
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

func (t *Tracker) SubscribeConnectedClients() (chan int, func()) {
	return t.trackBroadcaster.SubscribeClientCount(1)
}

// GetCurrentTrack Récupère la track actuelle et l'envoie dans le channel
func (t *Tracker) GetCurrentTrack() *model.Track {
	track, err := t.repo.FindLastTrack()
	if err != nil {
		t.log.Error("Failed to retrieve current track", "err", err)
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

// StartTracking Démarre le tracking des tracks, en supervisant la lecture de l'historique et en écoutant les tracks en direct pour alimenter les channels des clients
func (t *Tracker) StartTracking(ctx context.Context) {
	go t.superviseHistoryReader(ctx)
	go t.listenHistory(ctx)
}

// superviseHistoryReader supervise la lecture de l'historique des tracks.
// En cas de crash du lecteur, il attend deux secondes avant de le relancer.
// Le lecteur est arrêté lorsque le contexte est annulé.
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

// handleHistoryReader récupère la dernière track enregistrée puis les tracks passées jusqu'à la dernière track enregistrée.
// Les tracks passées et non enregistrées sont sauvegardées. Si la dernière track passée n'est pas finie, elle est envoyée dans le channel des tracks.
// Ensuite le suivi en direct des tracks est lancé.
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

// processTracks traites toutes les tracks passées non enregistrées, en sauvegardant dans l'historique
// les tracks finies et en envoyant dans le channel la track en cours si elle n'est pas terminée
func (t *Tracker) processTracks(tracks []*model.Track) {
	t.saveHistoryTracks(tracks[:len(tracks)-1])
	t.handleLastTrack(tracks[len(tracks)-1])
}

// saveHistoryTracks sauvegarde toutes les tracks
func (t *Tracker) saveHistoryTracks(tracks []*model.Track) {
	for _, track := range tracks {
		t.repo.AddTrackToHistory(track)
	}
}

// handleLastTrack vérifie si la dernière track est finie ou non.
// Si elle est finie, elle est sauvegardée dans l'historique.
// Sinon, elle est envoyée dans le channel des tracks en direct et sera sauvegardée dans listenHistory
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
