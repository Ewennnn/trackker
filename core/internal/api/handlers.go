package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"trackker/internal/api/payloads"
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
			s.sendDisplayTrackEvent(sseW, current)
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
					return
				}
				flusher.Flush()
			case track := <-tracksChannel:
				s.sendDisplayTrackEvent(sseW, track)
			}
		}
	}
}

func (s *Server) ListenForControlSupervisionSSE(appctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		tracksChannel, unsubscribeTracks := s.tracker.SubscribeForTracks()
		defer unsubscribeTracks()
		clientsChannel, unsubscribeClients := s.tracker.SubscribeConnectedClients()
		defer unsubscribeClients()

		sseW := &Sse{w}

		if err := s.sendSupervisionHTTPOnline(sseW, true); err != nil {
			s.log.Error("Failed to send supervision snapshot", "err", err)
			return
		}

		initialConnectedClients := <-clientsChannel
		if err := s.sendSupervisionConnectedClients(sseW, initialConnectedClients); err != nil {
			s.log.Error("Failed to send connected clients state", "err", err)
			return
		}

		if err := s.sendSupervisionCurrentTrack(sseW, r, s.tracker.GetCurrentTrack()); err != nil {
			s.log.Error("Failed to send current track state", "err", err)
			return
		}
		flusher.Flush()

		pingTicker := time.NewTicker(1 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-appctx.Done():
				s.log.Info("App context done, closing control supervision SSE")
				return
			case <-pingTicker.C:
				if _, err := sseW.Ping(); err != nil {
					s.log.Error("Failed to send supervision ping", "err", err)
					return
				}
				flusher.Flush()
			case connectedClients := <-clientsChannel:
				if err := s.sendSupervisionConnectedClients(sseW, connectedClients); err != nil {
					s.log.Error("Failed to send connected clients update", "err", err)
					return
				}
				flusher.Flush()
			case track := <-tracksChannel:
				if err := s.sendSupervisionCurrentTrack(sseW, r, track); err != nil {
					s.log.Error("Failed to send current track update", "err", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

func (s *Server) formatAndSendIcon(sseW *Sse) {
	fakeTrack := &model.Track{
		ID:   -1,
		Name: "Evyntia", // TODO replace by trackker logo
	}
	s.sendDisplayTrackEvent(sseW, fakeTrack)
}

func (s *Server) sendDisplayTrackEvent(sseW *Sse, track *model.Track) {
	response, err := s.formatter.Format(track)
	if err != nil {
		s.log.Error("Failed to format cover data", "err", err)
		return
	}

	if err := sseW.SendEvent("track", response); err != nil {
		s.log.Error("Failed to send response", "err", err)
	}
}

func (s *Server) sendSupervisionEvent(sseW *Sse, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return sseW.SendEvent(event, string(encoded))
}

func (s *Server) sendSupervisionHTTPOnline(sseW *Sse, online bool) error {
	return s.sendSupervisionEvent(sseW, "http_online", map[string]bool{
		"httpOnline": online,
	})
}

func (s *Server) sendSupervisionConnectedClients(sseW *Sse, count int) error {
	return s.sendSupervisionEvent(sseW, "connected_clients", map[string]int{
		"connectedClients": count,
	})
}

func (s *Server) sendSupervisionCurrentTrack(sseW *Sse, r *http.Request, track *model.Track) error {
	return s.sendSupervisionEvent(sseW, "current_track", map[string]*payloads.SupervisionTrackPayload{
		"currentTrack": payloads.BuildSupervisionTrackPayload(r, track),
	})
}
