package display

import (
	"net/http"
	"time"
	"trackker/internal/api"
	"trackker/internal/model"
	"trackker/internal/utils"
)

func (s *Server) LoadIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	}
}

func (s *Server) GetCover() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "-1" {
			// TODO replace by trackker logo
			http.ServeFile(w, r, "static/evyntia.svg")
			return
		}

		current := s.tracker.GetCurrentTrack()
		if current == nil {
			http.ServeFile(w, r, "static/evyntia.svg")
			return
		}

		cover := utils.GetTrackCover(current.Path)
		if cover == nil {
			http.ServeFile(w, r, "static/evyntia.svg")
			return
		}

		w.Header().Set("Content-Type", cover.MIMEType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cover.Data)
	}
}

func (s *Server) ListenForTracksSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		}

		tracksChannel, unsubscribe := s.tracker.SubscribeForTracks()
		defer unsubscribe()
		modeChannel, unsubscribeMode := s.controls.SubscribeDisplayMode() // TODO Voir pour bouger la gestion du display mode autre part (service commun entre tracker et controls)
		defer unsubscribeMode()

		sseW := &api.Sse{ResponseWriter: w}

		if err := s.sendDisplayForMode(sseW, s.controls.GetDisplayMode()); err != nil {
			s.log.Error("Failed to send initial display payload", "err", err)
			return
		}
		flusher.Flush()

		ping := time.NewTicker(1 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-r.Context().Done():
				s.formatAndSendIcon(sseW)
				return
			case mode := <-modeChannel:
				if err := s.sendDisplayForMode(sseW, mode); err != nil {
					s.log.Error("Failed to send forced display payload", "err", err)
					return
				}
				flusher.Flush()
			case <-ping.C:
				if _, err := sseW.Ping(); err != nil {
					s.log.Error("Failed to send ping", "err", err)
					return
				}
				flusher.Flush()
			case track := <-tracksChannel:
				if s.controls.GetDisplayMode() != model.DisplayModeLive {
					continue
				}

				s.sendDisplayTrackEvent(sseW, track)
				flusher.Flush()
			}
		}
	}
}

func (s *Server) sendDisplayTrackEvent(sseW *api.Sse, track *model.Track) {
	response, err := s.formatter.Format(track)
	if err != nil {
		s.log.Error("Failed to format cover data", "err", err)
		return
	}

	if err := sseW.SendEvent("track", response); err != nil {
		s.log.Error("Failed to send response", "err", err)
	}
}

func (s *Server) formatAndSendIcon(sseW *api.Sse) {
	fakeTrack := &model.Track{
		ID:   -1,
		Name: "Evyntia", // TODO replace by trackker logo
	}
	s.sendDisplayTrackEvent(sseW, fakeTrack)
}

func (s *Server) sendDisplayForMode(sseW *api.Sse, mode model.DisplayMode) error {
	switch mode {
	case model.DisplayModeBlackout:
		return s.sendDisplayBlackoutEvent(sseW)
	case model.DisplayModeFreezeTracking:
		s.formatAndSendIcon(sseW)
		return nil
	default:
		if current := s.tracker.GetCurrentTrack(); current != nil {
			s.sendDisplayTrackEvent(sseW, current)
		} else {
			s.formatAndSendIcon(sseW)
		}
		return nil
	}
}

func (s *Server) sendDisplayBlackoutEvent(sseW *api.Sse) error {
	return sseW.SendEvent("track", `<div class="track-container blackout-screen" aria-hidden="true"></div>`)
}
