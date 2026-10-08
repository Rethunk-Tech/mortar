package main

import "testing"

func TestReplayEmitsQueuedOnce(t *testing.T) {
	var got []string
	s := &UIIntentService{emit: func(name string, _ any) { got = append(got, name) }}
	s.Queue("a", nil)
	s.Queue("b", 1)
	s.Replay()
	s.Replay()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("emitted %v, want [a b] once", got)
	}
}
