package main

import (
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

	db, err := database.Init(conf)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db, logger); err != nil {
		log.Fatal(err)
	}

	repo := repository.New(logger, db)
	if err := repo.PrepareEvent(); err != nil {
		log.Fatal(err)
	}

	trackerService := service.New(logger, conf, repo, tracksParser)
	trackerService.StartTracking()

	server := api.NewServer(conf, logger, trackerService, sseFormatter)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
