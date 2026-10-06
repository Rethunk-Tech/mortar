package control

import "testing"

func TestOpenRequestHandsTheTargetToTheWindow(t *testing.T) {
	t.Parallel()
	var got []string
	s := &Services{Handoff: func(args []string) { got = args }}
	if _, err := s.Handle(t.Context(), OpenRequestMethod, Params{Path: "/tmp/farm.mortar"}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "/tmp/farm.mortar" {
		t.Fatalf("handoff = %v", got)
	}
	if _, err := s.Handle(t.Context(), OpenRequestMethod, Params{}); err == nil {
		t.Fatal("an open request without a target must be refused")
	}
	if _, err := (&Services{}).Handle(t.Context(), OpenRequestMethod, Params{Path: "x"}); err == nil {
		t.Fatal("an open request with no window to take it must be refused")
	}
}
