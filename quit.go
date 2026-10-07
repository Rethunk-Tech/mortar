package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const quitRequestedEvent = "quit:requested"

// quitGrace is how long the graceful quit (window and webview teardown, shutdown hooks) gets before Mortar finishes
// the shutdown itself and exits: a main thread stuck in a toolkit call must not keep a quit waiting for half a minute.
const quitGrace = 2 * time.Second

type QuitService struct {
	app *application.App
	// finish runs the shutdown work once, whichever path reaches it first.
	finish func()
	// busy reports work a forced exit must not cut off (an install writing into the store).
	busy    func() bool
	exit    func(int)
	grace   time.Duration
	watch   sync.Once
	queue   *queue.Service
	lan     *lan.Service
	launch  *launchsvc.Service
	mu      sync.Mutex
	allowed bool
}

func (s *QuitService) BusySummary() string {
	state := s.queue.State()
	for _, item := range state.Items {
		if item.State == queue.StateDownloading || item.State == queue.StateInstalling || item.State == queue.StateWaitingClick {
			return "Downloads and the running game will be interrupted"
		}
	}
	if s.lan.Busy() {
		return "Downloads and the running game will be interrupted"
	}
	if s.launch.AnyBusy() {
		return "Downloads and the running game will be interrupted"
	}
	return ""
}

//wails:ignore
func (s *QuitService) RequestQuit() {
	s.app.Event.Emit(quitRequestedEvent, s.BusySummary())
}

func (s *QuitService) ConfirmQuit() {
	s.mu.Lock()
	s.allowed = true
	s.mu.Unlock()
	s.armWatchdog()
	s.app.Quit()
}

// WatchSignals arms the watchdog on SIGINT and SIGTERM, which Wails turns into a quit on the main thread.
//
//wails:ignore
func (s *QuitService) WatchSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		s.armWatchdog()
	}()
}

// armWatchdog exits the process quitGrace after the first quit request unless the graceful path got there first,
// holding off while an install runs.
func (s *QuitService) armWatchdog() {
	s.watch.Do(func() {
		go func() {
			time.Sleep(s.grace)
			for s.busy() {
				time.Sleep(100 * time.Millisecond)
			}
			log.Printf("quit: graceful shutdown took over %s; finishing it directly", s.grace)
			s.finish()
			s.exit(0)
		}()
	})
}

//wails:ignore
func (s *QuitService) AllowWindowClose() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allowed {
		s.allowed = false
		return true
	}
	return false
}

// otherInstance reports a Mortar already serving dataDir. A server build has no single-instance lock, and a desktop
// build runs without one when the session bus is missing, so a second start reaches here and must stop before it
// rotates the log, recovers deploys or takes over the control file. A restart's previous process may still be exiting,
// so an answer is only trusted once it persists.
func otherInstance(dataDir string) (int, bool) {
	pid, ok := controlwire.Live(dataDir)
	for deadline := time.Now().Add(5 * time.Second); ok && time.Now().Before(deadline); {
		time.Sleep(200 * time.Millisecond)
		pid, ok = controlwire.Live(dataDir)
	}
	return pid, ok
}
