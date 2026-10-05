package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var _ application.ServiceStartup = (*startGate)(nil)

// startGate holds main-thread work an event listener sends before the app runs, when InvokeSync has no platform app
// to dispatch on and panics. Wails starts services once that app exists, so startup lets the held work through.
type startGate struct {
	mu      sync.Mutex
	started bool
	early   []func()
}

// run does fn on the main thread now, or once the app has started.
func (g *startGate) run(fn func()) {
	g.mu.Lock()
	if !g.started {
		g.early = append(g.early, fn)
		g.mu.Unlock()
		return
	}
	g.mu.Unlock()
	application.InvokeSync(fn)
}

func (g *startGate) ServiceStartup(context.Context, application.ServiceOptions) error {
	g.mu.Lock()
	g.started = true
	early := g.early
	g.early = nil
	g.mu.Unlock()
	for _, fn := range early {
		application.InvokeSync(fn)
	}
	return nil
}
