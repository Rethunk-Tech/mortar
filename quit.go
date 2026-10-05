package main

import (
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const quitRequestedEvent = "quit:requested"

type QuitService struct {
	app     *application.App
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
	s.app.Quit()
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
