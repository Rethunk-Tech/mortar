package settings

import "testing"

func TestSetLastModUpdateDigest(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	if len(s.Get().LastModUpdateDigest) != 0 {
		t.Fatalf("default digest = %v", s.Get().LastModUpdateDigest)
	}
	keys := []string{"a\x00" + "1.0"}
	at := "2026-10-03T12:00:00Z"
	if err := svc.SetLastModUpdateDigest(keys, at); err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if len(got.LastModUpdateDigest) != 1 || got.LastModUpdateDigest[0] != keys[0] || got.LastModUpdateDigestAt != at {
		t.Fatalf("stored = %#v at %q", got.LastModUpdateDigest, got.LastModUpdateDigestAt)
	}
}
