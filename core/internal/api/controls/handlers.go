package controls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"trackker/internal/api"
	"trackker/internal/model"
)

func (s *Server) GetStreamDeck() api.JsonResponseHandlerFunc {
	return func(r *http.Request) (any, int, error) {
		buttons, err := s.controls.GetStreamDeck()
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		return buttons, http.StatusOK, nil
	}
}

func (s *Server) SaveStreamDeckButton() api.JsonRequestResponseHandlerFunc[model.ButtonPayload] {
	return func(r *http.Request, body model.ButtonPayload) (any, int, error) {
		button, err := s.controls.SaveStreamDeckButton(body)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		return button, http.StatusCreated, nil
	}
}

type currentDisplayModePayload struct {
	DisplayMode model.DisplayMode `json:"displayMode"`
}

func (s *Server) GetCurrentDisplayMode() api.JsonResponseHandlerFunc {
	return func(request *http.Request) (any, int, error) {
		displayMode := s.controls.GetDisplayMode()
		return currentDisplayModePayload{DisplayMode: displayMode}, http.StatusOK, nil
	}
}

func (s *Server) ClickOnButton() api.JsonResponseHandlerFunc {
	return func(r *http.Request) (any, int, error) {
		buttonID := r.PathValue("id")
		if buttonID == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("button id is required")
		}

		id, err := strconv.Atoi(buttonID)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("unable to parse id value %s", buttonID)
		}

		if err := s.controls.ClickOnButton(int64(id)); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("unable to find button with id %d", id)
		}

		fmt.Println("PASS HERE")
		return s.GetCurrentDisplayMode()(r)
	}
}

func (s *Server) RunDisplayServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started, err := s.controls.StartDisplayServer()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !started {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) StopDisplayServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stopped, err := s.controls.StopDisplayServer()
		if err != nil {
			s.log.Error("Failed to stop display server", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !stopped {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// TODO Rework this endpoint
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
		modeChannel, unsubscribeMode := s.controls.SubscribeDisplayMode()
		defer unsubscribeMode()

		sseW := &api.Sse{ResponseWriter: w}
		currentMode := <-modeChannel

		if err := s.sendSupervisionHTTPOnline(sseW, true); err != nil {
			s.log.Error("Failed to send supervision snapshot", "err", err)
			return
		}

		initialConnectedClients := <-clientsChannel
		if err := s.sendSupervisionConnectedClients(sseW, initialConnectedClients); err != nil {
			s.log.Error("Failed to send connected clients state", "err", err)
			return
		}

		if err := s.sendSupervisionDisplayMode(sseW, currentMode); err != nil {
			s.log.Error("Failed to send display mode state", "err", err)
			return
		}

		//if err := s.sendSupervisionCurrentTrack(sseW, r, s.getCurrentTrackForMode(currentMode)); err != nil {
		//	s.log.Error("Failed to send current track state", "err", err)
		//	return
		//}
		//flusher.Flush()

		pingTicker := time.NewTicker(1 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-appctx.Done():
				s.log.Info("App context done, closing control supervision SSE")
				return
			case mode := <-modeChannel:
				currentMode = mode
				if err := s.sendSupervisionDisplayMode(sseW, currentMode); err != nil {
					s.log.Error("Failed to send display mode update", "err", err)
					return
				}
				//if err := s.sendSupervisionCurrentTrack(sseW, r, s.getCurrentTrackForMode(currentMode)); err != nil {
				//	s.log.Error("Failed to send current track update after mode change", "err", err)
				//	return
				//}
				flusher.Flush()
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
				//if currentMode != DisplayModeLive {
				//	continue
				//}

				if err := s.sendSupervisionCurrentTrack(sseW, r, track); err != nil {
					s.log.Error("Failed to send current track update", "err", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

func (s *Server) sendSupervisionHTTPOnline(sseW *api.Sse, online bool) error {
	return s.sendSupervisionEvent(sseW, "http_online", map[string]bool{
		"httpOnline": online,
	})
}

func (s *Server) sendSupervisionDisplayMode(sseW *api.Sse, mode model.DisplayMode) error {
	return s.sendSupervisionEvent(sseW, "display_mode", map[string]model.DisplayMode{
		"displayMode": mode,
	})
}

func (s *Server) sendSupervisionCurrentTrack(sseW *api.Sse, r *http.Request, track *model.Track) error {
	return s.sendSupervisionEvent(sseW, "current_track", map[string]*model.SimpleTrackResponse{
		"currentTrack": model.BuildSupervisionTrackPayload(r, track),
	})
}

func (s *Server) sendSupervisionConnectedClients(sseW *api.Sse, count int) error {
	return s.sendSupervisionEvent(sseW, "connected_clients", map[string]int{
		"connectedClients": count,
	})
}

func (s *Server) sendSupervisionEvent(sseW *api.Sse, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return sseW.SendEvent(event, string(encoded))
}
