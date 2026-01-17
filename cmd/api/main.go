package main

import (
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
		tracker.StartTracking()

		server := api.NewServer(conf, logger, tracker, sseFormatter)
		return server.Start()
	})

	if err != nil {
		log.Panicln(err)
	}
}
