package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"trackker/internal/app"
)

func main() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		waitForSignals(appCtx, cancel, logger)
	}()

	trackker := app.NewApp(logger)
	if err := trackker.Start(appCtx, &wg); err != nil {
		logger.Error("App stopped with error", "err", err)
		cancel()
	}

	wg.Wait()
}

func waitForSignals(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		logger.Info("Signal received to close the app", "signal", sig)
		cancel()
	case <-ctx.Done():
		return
	}
}
