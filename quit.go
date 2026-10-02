package main

import (
	"sync"

	"github.com/Rethunk-AI/mortar/internal/lan"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/queue"
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
	status, err := s.launch.Status("stardew")
	if err == nil && (status.State == launchsvc.Launching || status.State == launchsvc.Running) {
		return "Downloads and the running game will be interrupted"
	}
	return ""
}

func (s *QuitService) RequestQuit() {
	s.app.Event.Emit(quitRequestedEvent, s.BusySummary())
}

func (s *QuitService) ConfirmQuit() {
	s.mu.Lock()
	s.allowed = true
	s.mu.Unlock()
	s.app.Quit()
}

func (s *QuitService) AllowWindowClose() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allowed {
		s.allowed = false
		return true
	}
	return false
}
