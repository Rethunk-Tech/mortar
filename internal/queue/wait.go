package queue

import (
	"context"
	"fmt"
	"time"
)

// AddAndWait queues reqs and returns once every queued item is installed, or the first one fails, is skipped or
// cancelled, or ctx ends.
func (s *Service) AddAndWait(ctx context.Context, reqs []Request) error {
	items, err := s.Add(ctx, reqs)
	if err != nil {
		return err
	}
	ids := make(map[string]bool, len(items))
	for _, it := range items {
		ids[it.ID] = true
	}
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		pending := 0
		for _, it := range s.State().Items {
			if !ids[it.ID] {
				continue
			}
			switch it.State {
			case StateDone:
			case StateFailed:
				return fmt.Errorf("%s: %s", it.Name, it.Error)
			case StateSkipped, StateCancelled:
				return fmt.Errorf("%s was %s", it.Name, it.State)
			default:
				pending++
			}
		}
		if pending == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
