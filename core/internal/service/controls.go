package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"trackker/internal/api"
	"trackker/internal/model"
	"trackker/internal/repository"
)

type Controls struct {
	appCtx context.Context
	log    *slog.Logger
	repo   *repository.Repository

	displayMode            model.DisplayMode
	displayModeMu          sync.RWMutex
	displayModeBroadcaster *Broadcaster[model.DisplayMode]

	displayServerBuilder       api.ServerBuilder
	displayServer              api.Server
	displayServerMu            sync.Mutex
	cancelDisplayServerContext context.CancelFunc
}

func NewControls(appCtx context.Context, log *slog.Logger, repo *repository.Repository, displayServerBuilder api.ServerBuilder) *Controls {
	return &Controls{
		appCtx:               appCtx,
		log:                  log,
		repo:                 repo,
		displayServerBuilder: displayServerBuilder,
		displayMode:          model.DisplayModeLive,

		displayModeBroadcaster: NewBroadcaster[model.DisplayMode](log),
	}
}

func (c *Controls) StartDisplayServer() (bool, error) {
	c.displayServerMu.Lock()
	defer c.displayServerMu.Unlock()

	if c.displayServer != nil {
		return false, nil
	}

	ds := c.displayServerBuilder()
	ctx, cancel := context.WithCancel(c.appCtx)

	c.cancelDisplayServerContext = cancel
	c.displayServer = ds
	go func() {
		err := ds.Start(ctx)

		c.displayServerMu.Lock()
		defer c.displayServerMu.Unlock()

		// Avoid concurrent effects if new server restarted and override the oldest server
		if c.displayServer == ds {
			c.displayServer = nil
			c.cancelDisplayServerContext = nil
		}

		if err != nil {
			c.log.Error("Display server stopped with error", "error", err)
		}
	}()

	return true, nil
}

func (c *Controls) StopDisplayServer() (bool, error) {
	c.displayServerMu.Lock()
	defer c.displayServerMu.Unlock()

	if c.displayServer == nil {
		return false, nil
	}

	c.cancelDisplayServerContext()

	return true, nil
}

func (c *Controls) GetDisplayServerStatus() bool {
	c.displayServerMu.Lock()
	defer c.displayServerMu.Unlock()

	return c.displayServer != nil
}

// GetStreamDeck returns all buttons stored in database that represent the whole controls stream deck
func (c *Controls) GetStreamDeck() ([]*model.Button, error) {
	return c.repo.GetAllButtons()
}

func (c *Controls) SaveStreamDeckButton(buttonPayload model.ButtonPayload) (model.Button, error) {
	button := buttonPayload.ToModel()
	if err := c.repo.SaveNewButton(button); err != nil {
		return model.Button{}, err
	}
	return *button, nil
}

func (c *Controls) ClickOnButton(buttonID int64) error {
	button, err := c.repo.GetButtonById(buttonID)
	if err != nil {
		return fmt.Errorf("button id %d is not found", buttonID)
	}

	if !button.DisplayMode.IsValid() {
		return fmt.Errorf("button id %d has invalid display mode: %s", button.ID, button.DisplayMode)
	}

	if c.displayMode == button.DisplayMode {
		c.displayMode = model.DisplayModeLive
	} else {
		c.displayMode = button.DisplayMode
	}

	c.displayModeBroadcaster.Broadcast(c.displayMode)
	return nil
}

func (c *Controls) GetDisplayMode() model.DisplayMode {
	c.displayModeMu.RLock()
	defer c.displayModeMu.RUnlock()

	return c.displayMode
}

func (c *Controls) SetDisplayMode(mode model.DisplayMode) {
	c.displayModeMu.Lock()
	c.displayMode = mode
	c.displayModeMu.Unlock()

	c.displayModeBroadcaster.Broadcast(mode)
}

func (c *Controls) SubscribeDisplayMode() (chan model.DisplayMode, func()) {
	return c.displayModeBroadcaster.Subscribe(1)
}
