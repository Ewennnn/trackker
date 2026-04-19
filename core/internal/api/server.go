package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"trackker/internal/api/formatter"
	"trackker/internal/config"
	"trackker/internal/service"

	"github.com/alexedwards/scs/v2"
)

type Server struct {
	config         *config.Config
	log            *slog.Logger
	tracker        *service.Tracker
	formatter      formatter.Formatter
	httpServer     *http.Server
	sessionManager *scs.SessionManager
	attempts       map[string]*Attempt
	mu             sync.Mutex
}

func createSessionManager() *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = "trackker:session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	return sessionManager
}

func NewServer(config *config.Config, log *slog.Logger, service *service.Tracker, formatter formatter.Formatter) *Server {
	return &Server{
		config:         config,
		log:            log,
		tracker:        service,
		formatter:      formatter,
		sessionManager: createSessionManager(),
		attempts:       make(map[string]*Attempt),
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
		Handler: s.sessionManager.LoadAndSave(corsMiddleware(mux)),
	}

	s.log.Info("Server listening on: " + s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
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
