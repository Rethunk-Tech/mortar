package main

import (
	"log/slog"
	"sync"
	"time"
)

// processStart is as close to the launch as Go code sees, so startup lines count from it.
var processStart = time.Now()

var (
	startupMu   sync.Mutex
	startupLast = processStart
)

// startupStep logs a startup milestone with the time since launch and since the previous milestone, so a user's
// mortar.log shows where a slow start went.
func startupStep(phase string) {
	startupMu.Lock()
	defer startupMu.Unlock()
	now := time.Now()
	slog.Info("startup: step done", "phase", phase, "sinceStart", now.Sub(processStart).Round(time.Millisecond), "sinceStep", now.Sub(startupLast).Round(time.Millisecond))
	startupLast = now
}

// since is a background job's duration for its log line.
func since(start time.Time) time.Duration { return time.Since(start).Round(time.Millisecond) }
