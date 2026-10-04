package queue

import (
	"sync"
	"testing"
	"time"
)

func TestProgressTickSendsOnlyTheDelta(t *testing.T) {
	var mu sync.Mutex
	var events []string
	var delta Progress
	changed := 0
	now := time.Unix(100, 0).UTC()
	s := &Service{
		d: Deps{
			Now: func() time.Time { return now },
			Emit: func(name string, data any) {
				mu.Lock()
				defer mu.Unlock()
				events = append(events, name)
				if p, ok := data.(Progress); ok {
					delta = p
				}
			},
			Changed: func(State) { changed++ },
		},
		items: []*Item{{ID: "a", State: StateDownloading}},
	}
	p := &progress{s: s, id: "a", total: 1000, last: now.Add(-time.Second)}
	p.set(500)

	if len(events) != 1 || events[0] != ProgressEvent || changed != 0 {
		t.Fatalf("tick events = %v, changed hooks = %d; want one %s and no full state", events, changed, ProgressEvent)
	}
	if delta.ID != "a" || delta.Progress != 50 || delta.Speed != 500 {
		t.Fatalf("delta = %+v", delta)
	}

	s.publish(false)
	if len(events) != 2 || events[1] != ChangedEvent || changed != 1 {
		t.Fatalf("transition events = %v, changed hooks = %d; want a full state", events, changed)
	}
}
