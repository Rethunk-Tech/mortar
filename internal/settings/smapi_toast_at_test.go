package settings

import "testing"

func TestSetSmapiToastAt(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	if s.Get().SmapiToastAt != "" {
		t.Fatalf("default = %q", s.Get().SmapiToastAt)
	}
	at := "2026-09-30T12:00:00Z"
	if err := svc.SetSmapiToastAt(at); err != nil || s.Get().SmapiToastAt != at {
		t.Fatalf("set: %v %q", err, s.Get().SmapiToastAt)
	}
}
