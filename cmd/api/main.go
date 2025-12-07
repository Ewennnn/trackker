package main

import (
	"djtracker/internal/api"
	"djtracker/internal/config"
	"djtracker/internal/database"
	"djtracker/internal/repository"
	"djtracker/internal/service"
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

	s := service.New(logger, conf, repo)
	if err := s.StartTracking(); err != nil {
		log.Fatal(err)
	}

	server := api.NewServer(conf, logger, s)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
