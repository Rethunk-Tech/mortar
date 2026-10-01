package updatesvc

import (
	"context"
	"log"
	"time"
)

const (
	backgroundInitial = 30 * time.Second
	backgroundEvery   = 4 * time.Hour
)

// StagedEvent is emitted when a background download finishes and Restart or quit can apply the update.
const StagedEvent = "update:staged"

// StartBackground checks for Mortar updates on a timer and stages them without user action.
func (s *Service) StartBackground(ctx context.Context, emit func(string, any)) {
	if s.info.Off != "" {
		return
	}
	go s.backgroundLoop(ctx, emit)
}

func (s *Service) backgroundLoop(ctx context.Context, emit func(string, any)) {
	timer := time.NewTimer(backgroundInitial)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.backgroundTick(ctx, emit)
			timer.Reset(backgroundEvery)
		}
	}
}

func (s *Service) backgroundTick(ctx context.Context, emit func(string, any)) {
	rel, err := s.Check(ctx)
	if err != nil || rel == nil {
		return
	}
	if rel.Staged {
		return
	}
	if err := s.Install(ctx); err != nil {
		log.Printf("updater background install: %v", err)
		return
	}
	s.lock()
	r := s.found
	s.mu.Unlock()
	if r != nil && emit != nil {
		emit(StagedEvent, *r)
	}
}
