package api

import (
	"context"
	"djtracker/internal/model"
	"djtracker/internal/utils"
	"net/http"
	"time"
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

func (s *Server) ListenForTracksSSE(appctx context.Context) http.HandlerFunc {
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

		sseW := &Sse{w}
		if current := s.tracker.GetCurrentTrack(); current != nil {
			s.formatAndSendSse(sseW, current)
		} else {
			s.formatAndSendIcon(sseW)
		}

		ping := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-appctx.Done():
				s.log.Info("App context done, closing SSE connection")
				s.formatAndSendIcon(sseW)
				return
			case <-ping.C:
				if _, err := sseW.Ping(); err != nil {
					s.log.Error("Failed to send ping", "err", err)
					continue
				}
				flusher.Flush()
			case track := <-tracksChannel:
				s.formatAndSendSse(sseW, track)
			}
		}
	}
}

func (s *Server) formatAndSendIcon(sseW *Sse) {
	fakeTrack := &model.Track{
		ID:   -1,
		Name: "Evyntia", // TODO replace by trackker logo
	}
	s.formatAndSendSse(sseW, fakeTrack)
}

func (s *Server) formatAndSendSse(sseW *Sse, track *model.Track) {
	response, err := s.formatter.Format(track)
	if err != nil {
		s.log.Error("Failed to format cover data", "err", err)
		return
	}

	if err := sseW.SendEvent("track", response); err != nil {
		s.log.Error("Failed to send response", "err", err)
	}
}
