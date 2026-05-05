package display

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
	"trackker/internal/api/formatter"
	"trackker/internal/service"
)

type Server struct {
	log         *slog.Logger
	formatter   formatter.Formatter
	multiplexer *service.Multiplexer

	httpServer *http.Server
}

func NewServer(log *slog.Logger, formatter formatter.Formatter, multiplexer *service.Multiplexer) *Server {
	return &Server{
		log:         log,
		formatter:   formatter,
		multiplexer: multiplexer,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	mux.Handle("GET /", s.LoadIndex())
	mux.Handle("GET /cover/{id}", s.GetCover())
	mux.Handle("GET /events", s.ListenDisplayEventsSSE())

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", "localhost", "9000"),
		Handler: mux,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		if err := s.Shutdown(); err != nil {
			s.log.Error("Failed to shutdown display server", "err", err)
		}
	}()

	s.log.Info("Display server listening on: " + s.httpServer.Addr)
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
