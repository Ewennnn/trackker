package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) LoadIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	}
}

func (s *Server) GetCover() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current := s.service.GetCurrentTrack()
		if current == nil || current.Cover == nil {
			http.NotFound(w, r)
			return
		}

		// Extraire le mime type depuis le data URL
		parts := strings.SplitN(*current.Cover, ",", 2)
		if len(parts) != 2 {
			http.Error(w, "Invalid cover data", http.StatusInternalServerError)
			return
		}
		meta, b64data := parts[0], parts[1]

		mime := strings.TrimPrefix(meta, "data:")
		mime = strings.TrimSuffix(mime, ";base64")

		imgBytes, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			http.Error(w, "Failed to decode image", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", mime)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgBytes)
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
			tmpl, err := s.formatter.Format(current)
			if err != nil {
				fmt.Println(err)
				tmpl = "<h1>Error</h1>"
			}
			if err := sseW.SendEvent("track", tmpl); err != nil {
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
				tmpl, err := s.formatter.Format(track)
				if err != nil {
					fmt.Println(err)
					tmpl = "<h1>Error</h1>"
				}
				if err := sseW.SendEvent("track", tmpl); err != nil {
					break
				}
			}
		}
	}
}
