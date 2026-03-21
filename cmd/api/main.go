package main

import (
	"context"
	"database/sql"
	"djtracker/internal/api"
	"djtracker/internal/api/formatter"
	"djtracker/internal/config"
	"djtracker/internal/database"
	"djtracker/internal/repository"
	"djtracker/internal/service"
	"djtracker/internal/service/parser"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	conf, err := config.New()
	if err != nil {
		log.Fatal(err)
	}
	if err := conf.Check(); err != nil {
		log.Fatal(err)
	}

	tracksParser, err := parser.GetParser(conf, logger)
	if err != nil {
		log.Fatal(err)
	}
	if err := tracksParser.CheckState(); err != nil {
		log.Fatal(err)
	}

	sseFormatter, err := formatter.NewFormatter(conf, logger)
	if err != nil {
		log.Fatal(err)
	}

	err = database.UseDb(conf, func(db *sql.DB) error {
		repo := repository.New(logger, db)
		if err := repo.PrepareEvent(); err != nil {
			return err
		}

		tracker := service.NewTracker(logger, conf, repo, tracksParser)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tracker.StartTracking(ctx)

		server := api.NewServer(conf, logger, tracker, sseFormatter)
		serverError := make(chan error, 1)
		go func() {
			serverError <- server.Start(ctx)
		}()

		return listenForSignals(serverError, func() error {
			logger.Info("Signal received to close the app")
			cancel()
			return server.Shutdown()
		})
	})

	if err != nil {
		log.Panicln(err)
	}
}

func listenForSignals(serverError chan error, callback func() error) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-serverError:
		return callback()
	case <-sigChan:
		return callback()
	}
}
