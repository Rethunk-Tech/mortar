package savessvc

import (
	"testing"
	"time"
)

func TestStoreRecordGetAndOverwrite(t *testing.T) {
	s := NewStore(t.TempDir())
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := s.Record("stardew", "Farm_1", "p1", at); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Get("stardew", "Farm_1")
	if err != nil || !ok || got.ProfileID != "p1" || !got.At.Equal(at) {
		t.Fatalf("got %#v %v %v", got, ok, err)
	}
	later := at.Add(time.Hour)
	if err := s.Record("stardew", "Farm_1", "p2", later); err != nil {
		t.Fatal(err)
	}
	all, err := s.All("stardew")
	if err != nil || all["Farm_1"].ProfileID != "p2" || !all["Farm_1"].At.Equal(later) {
		t.Fatalf("all = %#v, %v", all, err)
	}
	if _, ok, err := s.Get("stardew", "Missing"); err != nil || ok {
		t.Fatalf("missing: %v %v", ok, err)
	}
}

func TestStoreRejectsEmptyKeys(t *testing.T) {
	s := NewStore(t.TempDir())
	if err := s.Record("", "Farm_1", "p1", time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestNotePlayedAndAttachLast(t *testing.T) {
	s := &Service{last: NewStore(t.TempDir())}
	s.NotePlayed("stardew", "p1", "Farm_1")
	fit := Fit{Folder: "Farm_1"}
	s.fillLast("stardew", &fit, nil, nil)
	if fit.LastProfileID != "p1" || fit.LastProfileAt == 0 {
		t.Fatalf("fit = %#v", fit)
	}
}
