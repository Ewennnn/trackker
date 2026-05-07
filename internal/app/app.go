package app

import (
	"context"
	"log/slog"
	"sync"
	"trackker/internal/api"
	"trackker/internal/api/controls"
	"trackker/internal/api/display"
	"trackker/internal/api/formatter"
	"trackker/internal/config"
	"trackker/internal/database"
	"trackker/internal/repository"
	"trackker/internal/service"
	"trackker/internal/service/parser"
	"trackker/internal/utils"
)

type Trackker struct {
	logger      *slog.Logger
	config      *config.Config
	tracker     *service.Tracker
	controls    *service.Controls
	multiplexer *service.Multiplexer
	repo        *repository.Repository
	parser      parser.Parser
	formatter   formatter.Formatter

	controlsServer *controls.Server
}

func NewApp(logger *slog.Logger) *Trackker {
	return &Trackker{
		logger: logger,
	}
}

func (app *Trackker) Start(appCtx context.Context, wg *sync.WaitGroup) error {
	var err error

	conf, err := config.New()
	if err != nil {
		return err
	}
	if err := conf.Check(); err != nil {
		return err

	}
	app.config = conf

	tracksParser, err := parser.GetParser(app.config, app.logger)
	if err != nil {
		return err
	}
	if err := tracksParser.CheckState(); err != nil {
		return err
	}
	app.parser = tracksParser

	responseFormatter, err := formatter.NewFormatter(app.config, app.logger)
	if err != nil {
		return err
	}
	app.formatter = responseFormatter

	db, err := database.Connect(app.config)
	if err != nil {
		return err
	}
	defer utils.SafeClose(db)
	if err := database.Migrate(db); err != nil {
		return err
	}

	repo := repository.New(app.logger, db)
	if err := repo.PrepareEvent(); err != nil {
		return err
	}
	app.repo = repo

	eventBus := make(chan service.Event, 1)

	controlsService := service.NewControls(appCtx, app.logger, repo, eventBus, app.buildDisplayServer)
	app.controls = controlsService

	trackerService := service.NewTracker(app.logger, app.config, repo, app.parser, eventBus)
	trackerService.StartTracking(appCtx, wg)
	app.tracker = trackerService

	multiplexer := service.NewMultiplexer(app.logger, eventBus)
	multiplexer.Init(trackerService, controlsService)
	multiplexer.Run(appCtx, wg)
	app.multiplexer = multiplexer

	controlsServer := controls.NewServer(app.logger, controlsService, multiplexer, app.config.Control.PinCode)
	app.controlsServer = controlsServer

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.controlsServer.Start(appCtx); err != nil {
			app.logger.Error("Failed to start controls server", "err", err)
		}
	}()

	app.logger.Info("App started successfully")
	<-appCtx.Done()
	return nil
}

func (app *Trackker) buildDisplayServer() api.Server {
	return display.NewServer(app.logger, app.formatter, app.multiplexer)
}
