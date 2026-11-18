package savessvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/saves"
)

func TestFarmOf(t *testing.T) {
	ctx := t.Context()
	s := &Service{}
	if got := s.farmOf(ctx, "stardew-valley", "Farm"); got != "Farm" {
		t.Fatalf("nil scanner: got %q", got)
	}
	if got := s.farmOf(ctx, "stardew-valley", ""); got != "" {
		t.Fatalf("empty folder: got %q", got)
	}

	dir := t.TempDir()
	s.scanners = map[string]*saves.Scanner{
		"g": {Dir: dir, CacheDir: t.TempDir()},
	}
	if got := s.farmOf(ctx, "g", "Missing"); got != "Missing" {
		t.Fatalf("empty scan: got %q", got)
	}

	file := filepath.Join(t.TempDir(), "not-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.scanners["g"] = &saves.Scanner{Dir: file, CacheDir: t.TempDir()}
	if got := s.farmOf(ctx, "g", "Farm"); got != "Farm" {
		t.Fatalf("scan err: got %q", got)
	}
}

func TestFarmFrom(t *testing.T) {
	infos := []saves.Info{
		{Folder: "A", Farm: "Alpha"},
		{Folder: "B"},
	}
	if got := farmFrom(infos, "A"); got != "Alpha" {
		t.Fatalf("named farm: got %q", got)
	}
	if got := farmFrom(infos, "B"); got != "B" {
		t.Fatalf("empty farm: got %q", got)
	}
	if got := farmFrom(infos, "C"); got != "C" {
		t.Fatalf("missing: got %q", got)
	}
}

func TestDescribeLacks(t *testing.T) {
	s := &Service{}
	if got := s.describeLacks(t.Context(), "stardew-valley", nil); len(got) != 0 {
		t.Fatalf("empty: got %v", got)
	}
}

func TestWantedIDs(t *testing.T) {
	id := mod.ID("smapi:A.B")
	got := wantedIDs([]Lack{{ID: id}})
	if !got[id] || len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	if n := wantedIDs(nil); len(n) != 0 {
		t.Fatalf("empty: %v", n)
	}
}

func TestFillLackNames(t *testing.T) {
	id := mod.ID("smapi:A.B")
	where := &problems.Ref{Site: "Nexus"}
	got := fillLackNames([]Lack{{ID: id, Name: "old"}}, map[mod.ID]described{
		id: {name: "Named", where: where},
	})
	if got[0].Name != "Named" || got[0].Where != where {
		t.Fatalf("filled: %+v", got[0])
	}
	keep := fillLackNames([]Lack{{ID: id, Name: "keep"}}, map[mod.ID]described{})
	if keep[0].Name != "keep" || keep[0].Where != nil {
		t.Fatalf("missing: %+v", keep[0])
	}
}

func TestCheckEdges(t *testing.T) {
	ctx := t.Context()
	s := &Service{}
	if _, err := s.Check(ctx, "g", "", ""); err == nil {
		t.Fatal("empty save: want error")
	}
	got, err := s.Check(ctx, "g", "Farm", "")
	if err != nil || got.Folder != "Farm" || got.Farm != "Farm" {
		t.Fatalf("last nil: %+v err=%v", got, err)
	}

	s.last = NewStore(t.TempDir())
	got, err = s.Check(ctx, "g", "Farm", "")
	if err != nil || got.LastProfileID != "" || got.Folder != "Farm" {
		t.Fatalf("no record: %+v err=%v", got, err)
	}

	bad := t.TempDir()
	gameDir := filepath.Join(bad, "g")
	if err := os.MkdirAll(gameDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "last-played.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.last = NewStore(bad)
	if _, err := s.Check(ctx, "g", "Farm", ""); err == nil {
		t.Fatal("corrupt last-played: want error")
	}
}

func TestCheckProfileMods(t *testing.T) {
	e := newSaveEnv(t)
	p := e.mainWithAlpha(t)
	last := NewStore(t.TempDir())
	if err := last.RecordRun("stardew", "Farm_1", p.ID, time.Unix(1, 0), []PlayedMod{
		{ID: "smapi:A.Mod", Name: "Alpha"},
	}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{last: last, profiles: e.profiles}

	if _, err := svc.Check(t.Context(), "stardew", "Farm_1", "missing-profile"); err == nil {
		t.Fatal("missing profile: want error")
	}

	got, err := svc.Check(t.Context(), "stardew", "Farm_1", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LastProfileExists || len(got.Missing) != 0 {
		t.Fatalf("check %+v", got)
	}
}
