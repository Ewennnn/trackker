package display

import (
	"net/http"
	"time"
	"trackker/internal/api"
	"trackker/internal/model"
	"trackker/internal/service"
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

		current := s.multiplexer.GetCurrentTrack()
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

func (s *Server) ListenDisplayEventsSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		sseW := &api.Sse{ResponseWriter: w}

		events, unsubscribe := s.multiplexer.SubscribeToDisplay()
		defer unsubscribe()

		ping := time.NewTicker(1 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-r.Context().Done():
				s.sendIconAsTrack(sseW)
				return
			case <-ping.C:
				sseW.Ping()
			case event := <-events:
				s.processDisplayEvent(event, sseW)
			}
		}
	}
}

func (s *Server) processDisplayEvent(event service.DisplayEvent, sseW *api.Sse) {
	switch evt := event.(type) {
	case service.TrackChangeEvent:
		s.sendDisplayTrackEvent(sseW, evt.Track)
	case service.DisplayModeChangeEvent:
		if evt.Mode == model.DisplayModeBlackout {
			s.sendBlackoutEvent(sseW)
		} else if evt.Mode == model.DisplayModeFreeze {
			s.sendIconAsTrack(sseW)
		}
	}
}

func (s *Server) sendDisplayModeEvent(sseW *api.Sse, mode model.DisplayMode) {
	switch mode {
	case model.DisplayModeBlackout:
		s.sendBlackoutEvent(sseW)
	case model.DisplayModeFreeze:
		s.sendIconAsTrack(sseW)
	}
}

func (s *Server) sendDisplayTrackEvent(sseW *api.Sse, track model.Track) {
	response, err := s.formatter.Format(track)
	if err != nil {
		s.log.Error("Failed to format cover data", "err", err)
		return
	}

	if err := sseW.SendEvent("track", response); err != nil {
		s.log.Error("Failed to send response", "err", err)
	}
}

func (s *Server) sendIconAsTrack(sseW *api.Sse) {
	fakeTrack := model.Track{
		ID:   -1,
		Name: "Evyntia", // TODO replace by trackker logo
	}
	s.sendDisplayTrackEvent(sseW, fakeTrack)
}

func (s *Server) sendBlackoutEvent(sseW *api.Sse) {
	if err := sseW.SendEvent("track", `<div class="track-container blackout-screen" aria-hidden="true"></div>`); err != nil {
		s.log.Error("Failed to send blackout event", "err", err)
	}
}
