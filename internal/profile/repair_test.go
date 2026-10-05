package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestRepairRebuildsFromSnapshotAndKeepsMods(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	dir := filepath.Join(e.root, "stardew", p.ID)
	marker := filepath.Join(dir, "mods", "keep-me")
	if err := os.WriteFile(marker, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := []byte(`{"name":"Farm"}x`)
	if err := os.WriteFile(filepath.Join(dir, fileName), broken, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := e.Repair("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Farm" || len(got.Entries) != 1 || got.Entries[0].Key != "local-a" {
		t.Fatalf("repaired = %+v", got)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("mods kept: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, damagedAsidePrefix+"*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("aside = %v, %v", matches, err)
	}
	aside, err := fsx.ReadFile(matches[0])
	if err != nil || string(aside) != string(broken) {
		t.Fatalf("aside body = %q, %v", aside, err)
	}
	if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
		t.Fatal(err)
	}
}

func TestRepairNoSnapshot(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	id := "0123456789abcdef"
	dir := filepath.Join(s.root, "stardew", id)
	if err := os.MkdirAll(filepath.Join(dir, "mods"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := s.Repair("stardew", id)
	if err == nil || !strings.Contains(err.Error(), "no history snapshot") {
		t.Fatalf("Repair = %v", err)
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, fileName))
	if err != nil || string(raw) != "{" {
		t.Fatalf("left damaged file: %q %v", raw, err)
	}
}

func TestRepairKeepsDamagedFileAsideAndUndoRestoresIt(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p := addFarmMod(t, e)
	dir := filepath.Join(e.root, "stardew", p.ID)
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Repair("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.UndoRepair("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, fileName))
	if err != nil || string(raw) != "{" {
		t.Fatalf("restored = %q %v", raw, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, damagedAsidePrefix+"*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("aside after undo = %v, %v", matches, err)
	}
}

func TestListDamagedSetsRepairErrorWithoutSnapshot(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	id := "0123456789abcdef"
	dir := filepath.Join(s.root, "stardew", id)
	if err := os.MkdirAll(filepath.Join(dir, "mods"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListDamaged("stardew")
	if err != nil || len(got) != 1 || got[0].RepairError == "" {
		t.Fatalf("ListDamaged = %+v, %v", got, err)
	}
}
