package shortcut

import "testing"

func TestPlayModeEndsWhenItsGameCloses(t *testing.T) {
	var shown []WindowMode
	s := &Service{Window: func(m WindowMode) { shown = append(shown, m) }}
	if !s.StartSolo([]string{"--play=stardew/abc"}) || !s.PlayMode() {
		t.Fatal("a --play start is play mode")
	}
	if r := s.Take(); r == nil || r.Profile != "abc" {
		t.Fatalf("the request is still handed to the window: %+v", r)
	}
	if s.Observe("stardew", false) {
		t.Fatal("an idle game that never ran does not end play mode")
	}
	if s.Observe("lethal-company", true) || s.Observe("lethal-company", false) {
		t.Fatal("another game does not end play mode")
	}
	if s.Observe("stardew", true) || !s.Observe("stardew", false) {
		t.Fatal("play mode ends when its game stops running")
	}
	s.ShowPrompt()
	s.HidePrompt()
	s.LeavePlayMode()
	s.ShowPrompt()
	if s.PlayMode() || s.Observe("stardew", false) {
		t.Fatal("after leaving, Mortar stays open when the game closes")
	}
	if len(shown) != 3 || shown[0] != WindowPrompt || shown[1] != WindowHidden || shown[2] != WindowFull {
		t.Fatalf("windows %v", shown)
	}
	if (&Service{}).StartSolo([]string{"mortar://x"}) {
		t.Fatal("no request, no play mode")
	}
}
