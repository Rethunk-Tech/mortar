package updatesvc

import (
	"context"
	"testing"
)

func TestWhatsNewOfflineKeepsLastRun(t *testing.T) {
	dir := t.TempDir()
	if err := writeLastRun(dir, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	s := &Service{info: Info{Version: "1.1.0"}, dir: dir}
	got, err := s.WhatsNew(context.Background())
	if err == nil || got.Version != "" {
		t.Fatalf("offline skip: %+v, %v", got, err)
	}
	last, err := readLastRun(dir)
	if err != nil || last != "1.0.0" {
		t.Fatalf("last run kept: %q, %v", last, err)
	}
}
