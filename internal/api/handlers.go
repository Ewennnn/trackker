package api

import (
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

		sseW := &SseWriter{w}
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
				trackJson, err := json.Marshal(track)
				if err != nil {
					s.log.Error("Error while converting struct to JSON", err)
					continue
				}

				packet := &SsePacket{
					Event: "track",
					Data:  string(trackJson),
				}
				if err := sseW.WritePacket(packet); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					break
				}
				flusher.Flush()
			}
		}
	}
}
