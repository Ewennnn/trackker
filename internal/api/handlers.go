package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) LoadIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	}
}

func (s *Server) StartSSE() http.HandlerFunc {
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

		sseW := &SseWriterInterceptor{w}
		for {
			select {
			case <-r.Context().Done():
				return
			case track := <-tracksChannel:

				s.log.Info("New track", "track", track)
				trackJson, err := json.Marshal(track)
				if err != nil {
					s.log.Error("Error while converting struct to JSON", err)
					continue
				}

				if _, err := fmt.Fprintf(sseW, string(trackJson)); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					break
				}
				flusher.Flush()
			}
		}
	}
}
