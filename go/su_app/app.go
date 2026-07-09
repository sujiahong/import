package su_app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.local/su_errors"
)

type App struct {
	name    string
	mu      sync.Mutex
	modules []Module
	started []Module
}

func New(name string) *App {
	return &App{name: name}
}

func (a *App) Name() string {
	if a == nil {
		return ""
	}
	return a.name
}

func (a *App) Register(module Module) {
	if a == nil || module == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.modules = append(a.modules, module)
}

func (a *App) Start(ctx context.Context) error {
	if a == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "nil app")
	}
	a.mu.Lock()
	modules := append([]Module(nil), a.modules...)
	a.started = a.started[:0]
	a.mu.Unlock()

	for _, module := range modules {
		if err := module.Start(ctx); err != nil {
			_ = a.Stop(ctx)
			return su_errors.Wrap(su_errors.CodeInternal, fmt.Sprintf("start module %s", module.Name()), err)
		}
		a.mu.Lock()
		a.started = append(a.started, module)
		a.mu.Unlock()
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	started := append([]Module(nil), a.started...)
	a.started = nil
	a.mu.Unlock()

	var errs []error
	for i := len(started) - 1; i >= 0; i-- {
		if err := started[i].Stop(ctx); err != nil {
			errs = append(errs, su_errors.Wrap(su_errors.CodeInternal, fmt.Sprintf("stop module %s", started[i].Name()), err))
		}
	}
	return errors.Join(errs...)
}
