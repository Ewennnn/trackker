package controls

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
	"trackker/internal/api"
	"trackker/internal/model"
	"trackker/internal/service"
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

func (s *Server) ListenForControlSupervisionSSE(appctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		events, unsubscribe := s.multiplexer.SubscribeToControls()
		defer unsubscribe()

		sseW := &api.Sse{ResponseWriter: w}
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
				sseW.Ping()
			case event := <-events:
				s.processControlsEvent(event, sseW)
			}
		}
	}
}

func (s *Server) processControlsEvent(event service.DisplayEvent, sseW *api.Sse) {
	switch evt := event.(type) {
	case service.TrackChangeEvent:
		s.sendSupervisionCurrentTrack(sseW, evt.Track)
	case service.DisplayModeChangeEvent:
		s.sendSupervisionDisplayMode(sseW, evt.Mode)
	case service.DisplayServerStatusChangeEvent:
		s.sendSupervisionHTTPOnline(sseW, evt.Running)
	case service.ConnectedClientEvent:
		s.sendSupervisionConnectedClients(sseW, evt.Service, evt.Count)
	}
}

func (s *Server) sendSupervisionHTTPOnline(sseW *api.Sse, online bool) {
	s.sendSupervisionEvent(sseW, "http_online", map[string]bool{
		"httpOnline": online,
	})
}

func (s *Server) sendSupervisionDisplayMode(sseW *api.Sse, mode model.DisplayMode) {
	s.sendSupervisionEvent(sseW, "display_mode", map[string]model.DisplayMode{
		"displayMode": mode,
	})
}

func (s *Server) sendSupervisionCurrentTrack(sseW *api.Sse, track model.Track) {
	s.sendSupervisionEvent(sseW, "current_track", map[string]model.SimpleTrackResponse{
		"currentTrack": model.BuildSupervisionTrackPayload(track),
	})
}

func (s *Server) sendSupervisionConnectedClients(sseW *api.Sse, service string, count int) {
	s.sendSupervisionEvent(sseW, "connected_clients", map[string]int{
		service: count,
	})
}

func (s *Server) sendSupervisionEvent(sseW *api.Sse, event string, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to encode supervision event payload", "err", err)
		return
	}

	if err := sseW.SendEvent(event, string(encoded)); err != nil {
		s.log.Error("Failed to send supervision event", "err", err)
	}
}

type TrackkerIPsResponse struct {
	Display  []string `json:"display"`
	Controls []string `json:"controls"`
}

func (s *Server) GetLocalIP() api.JsonResponseHandlerFunc {
	return func(r *http.Request) (any, int, error) {
		const (
			displayPort  = 9000
			controlsPort = 8080
		)

		ips, err := getLocalIPv4s()
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		return TrackkerIPsResponse{
			Display:  buildURLs(ips, displayPort),
			Controls: buildURLs(ips, controlsPort),
		}, http.StatusOK, nil
	}
}

// getLocalIPv4s récupère toutes les IPv4 uniques (inclut localhost)
func getLocalIPv4s() ([]string, error) {
	ipSet := make(map[string]struct{})

	// Toujours inclure localhost
	ipSet["localhost"] = struct{}{}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipnet.IP.To4()
			if ip == nil {
				continue
			}

			ipSet[ip.String()] = struct{}{}
		}
	}

	// Convert map → slice
	ips := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ips = append(ips, ip)
	}

	return ips, nil
}

// buildURLs construit les URLs pour un port donné
func buildURLs(ips []string, port int) []string {
	urls := make([]string, 0, len(ips))
	for _, ip := range ips {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
	}
	return urls
}
