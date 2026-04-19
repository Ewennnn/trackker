package api

import (
	"context"
	"djtracker/internal/api/formatter"
	"djtracker/internal/config"
	"djtracker/internal/service"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	config     *config.Config
	log        *slog.Logger
	tracker    *service.Tracker
	formatter  formatter.Formatter
	httpServer *http.Server
}

func NewServer(config *config.Config, log *slog.Logger, service *service.Tracker, formatter formatter.Formatter) *Server {
	return &Server{
		config:    config,
		log:       log,
		tracker:   service,
		formatter: formatter,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Controls endpoints
	mux.Handle("POST /api/pincode/{code}", s.checkPinCode())

	// Display endpoints
	mux.Handle("GET /", s.LoadIndex())
	mux.Handle("GET /cover/{id}", s.GetCover())
	mux.Handle("GET /events", s.ListenForTracksSSE(ctx))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", s.config.Server.BindAddress, s.config.Server.Port),
		Handler: corsMiddleware(mux),
	}

	s.log.Info("Server listening on: " + s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	// Safe timeout to wait all requests to finish before shutting down the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
