package frosted

import (
	"testing"
)

type fakeHost struct {
	sets       [][2]int
	setCalls   int
	clearCalls int
}

func (f *fakeHost) SetBlur(w, h int) {
	f.setCalls++
	f.sets = append(f.sets, [2]int{w, h})
}

func (f *fakeHost) Clear() { f.clearCalls++ }

func TestBlurTracksTranslucencyAndResize(t *testing.T) {
	h := &fakeHost{}
	s := New(h)

	s.Resize(1280, 720)
	if h.setCalls != 0 || h.clearCalls != 0 {
		t.Fatalf("solid resize applied blur: sets=%d clears=%d", h.setCalls, h.clearCalls)
	}

	s.SetTranslucent(true)
	if h.setCalls != 1 || h.sets[0] != [2]int{1280, 720} {
		t.Fatalf("translucent on: sets=%v", h.sets)
	}

	s.Resize(1600, 900)
	if h.setCalls != 2 || h.sets[1] != [2]int{1600, 900} {
		t.Fatalf("resize: sets=%v", h.sets)
	}

	s.SetTranslucent(false)
	if h.clearCalls != 1 {
		t.Fatalf("translucent off: clears=%d", h.clearCalls)
	}

	s.Resize(800, 600)
	if h.setCalls != 2 || h.clearCalls != 1 {
		t.Fatalf("solid resize after off: sets=%d clears=%d", h.setCalls, h.clearCalls)
	}
}

func TestMissingProtocolStillStarts(t *testing.T) {
	s := Attach(nil, 1280, 720, true)
	s.Resize(640, 480)
	s.SetTranslucent(false)
}
