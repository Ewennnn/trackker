package controls

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"trackker/internal/api"
	"trackker/internal/api/assets"
	"trackker/internal/service"

	"github.com/alexedwards/scs/v2"
)

type Server struct {
	log         *slog.Logger
	controls    *service.Controls
	multiplexer *service.Multiplexer

	sessionManager *scs.SessionManager
	pinCode        string
	attempts       map[string]*Attempt
	mu             sync.Mutex
	httpServer     *http.Server
}

func createSessionManager() *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = "trackker_session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Secure = false
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	return sessionManager
}

func NewServer(log *slog.Logger, controls *service.Controls, multiplexer *service.Multiplexer, securityPinCode string) *Server {
	return &Server{
		log:         log,
		controls:    controls,
		multiplexer: multiplexer,

		pinCode:        securityPinCode,
		sessionManager: createSessionManager(),
		attempts:       make(map[string]*Attempt),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// UI
	mux.Handle("/", http.FileServer(http.FS(assets.WebControlsFS())))

	// Security
	mux.Handle("POST /api/pincode/{code}", s.checkPinCode())
	mux.Handle("GET /api/control/session", s.getSessionStatus())

	// Stream deck
	mux.Handle("GET /api/control/ui/streamdeck", s.authMiddleware(api.JsonResponseWrapper(s.GetStreamDeck())))
	mux.Handle("POST /api/control/ui/streamdeck/button", s.authMiddleware(api.JsonRequestResponseWrapper(s.SaveStreamDeckButton())))
	mux.Handle("GET /api/control/actions/button", s.authMiddleware(api.JsonResponseWrapper(s.GetCurrentDisplayMode())))
	mux.Handle("POST /api/control/actions/button/{id}", s.authMiddleware(api.JsonResponseWrapper(s.ClickOnButton())))

	// Supervision
	mux.Handle("GET /api/control/supervision/events", s.authMiddleware(s.ListenForControlSupervisionSSE(ctx)))

	// Controls
	mux.Handle("POST /api/control/display/start", s.authMiddleware(s.RunDisplayServer()))
	mux.Handle("POST /api/control/display/stop", s.authMiddleware(s.StopDisplayServer()))

	// Data
	mux.Handle("GET /api/control/ip", s.authMiddleware(api.JsonResponseWrapper(s.GetLocalIP())))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", "0.0.0.0", "8080"),
		Handler: s.sessionManager.LoadAndSave(corsMiddleware(mux)),
	}

	go func() {
		<-ctx.Done()
		if err := s.Shutdown(); err != nil {
			s.log.Error("Failed to shutdown controls server", "err", err)
		}
	}()

	s.log.Info("Controls server listening on: " + s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
