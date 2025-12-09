package api

import (
	"djtracker/internal/model"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) LoadIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
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

		tracksChannel, unsubscribe := s.service.SubscribeForTracks()
		defer unsubscribe()

		sseW := &Sse{w}
		if current := s.service.GetCurrentTrack(); current != nil {
			if err := sseW.SendEvent("track", parseTrack(current)); err != nil {
				return
			}
		}

		ping := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ping.C:
				if _, err := sseW.Ping(); err != nil {
					s.log.Error("Failed to send ping", err)
					continue
				}
				flusher.Flush()
			case track := <-tracksChannel:
				if err := sseW.SendEvent("track", parseTrack(track)); err != nil {
					break
				}
			}
		}
	}
}

func parseTrack(data *model.TrackDTO) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
