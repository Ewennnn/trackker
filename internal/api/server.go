package api

import (
	formatter2 "djtracker/internal/api/formatter"
	"djtracker/internal/config"
	"djtracker/internal/service"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	config    *config.Config
	log       *slog.Logger
	service   *service.Service
	formatter formatter2.Formatter
}

func NewServer(config *config.Config, log *slog.Logger, service *service.Service, formatter formatter2.Formatter) *Server {
	return &Server{
		config:    config,
		log:       log,
		service:   service,
		formatter: formatter,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	mux.Handle("GET /", s.LoadIndex())
	mux.Handle("GET /cover/", s.GetCover())
	mux.Handle("GET /events", s.ListenForTracksSSE())

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", s.config.Server.BindAddress, s.config.Server.Port),
		Handler: mux,
	}

	s.log.Info("Server listening on: " + server.Addr)
	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
