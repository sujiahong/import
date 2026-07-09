package su_app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := a.Start(runCtx); err != nil {
		return err
	}
	<-runCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return a.Stop(shutdownCtx)
}
