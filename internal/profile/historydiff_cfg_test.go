package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureHistoryConfigsStoresOnlyConfigJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	im := filepath.Join(dir, "mods", "k")
	if err := os.MkdirAll(filepath.Join(im, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(im, "saves"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(im, "config.json"), []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(im, "data", "big.bin"), []byte("huge"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(im, "saves", "slot"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Key: "k"}}
	captureHistoryConfigs(dir, "snap", entries)
	loaded := loadHistoryConfigs(dir, "snap", entries)
	got := loaded["k"]["config.json"]
	if string(got) != `{"a":1}` {
		t.Fatalf("captured config = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, "snap", "k", "data", "big.bin")); err == nil {
		t.Fatal("captured data/")
	}
	sum := sha256.Sum256([]byte(`{"a":1}`))
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, historyBlobsDir, hex.EncodeToString(sum[:]))); err != nil {
		t.Fatalf("missing blob: %v", err)
	}
}

func TestCaptureHistoryConfigsDedupesBlobs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	im := filepath.Join(dir, "mods", "k")
	if err := os.MkdirAll(im, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(im, "config.json"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries := []Entry{{Key: "k"}}
	captureHistoryConfigs(dir, "s1", entries)
	captureHistoryConfigs(dir, "s2", entries)
	blobs, err := os.ReadDir(filepath.Join(dir, historyFilesDir, historyBlobsDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 {
		t.Fatalf("blobs = %d, want 1", len(blobs))
	}
}

func TestWriteHistoryPrunesDroppedSnapshotFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	body := []byte("keep")
	sum := sha256.Sum256(body)
	keepHash := hex.EncodeToString(sum[:])
	dropHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := os.MkdirAll(filepath.Join(dir, historyFilesDir, historyBlobsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, historyBlobsDir, keepHash), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, historyBlobsDir, dropHash), []byte("drop"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, historyFilesDir, "old-layout", "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, "old-layout", "k", "config.json"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	s1, s2 := "snap-keep", "snap-drop"
	if err := os.MkdirAll(filepath.Join(dir, historyFilesDir, s1), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, historyFilesDir, s2), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, s1, historySnapshotIndex), []byte(`{"k":{"config.json":"`+keepHash+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, historyFilesDir, s2, historySnapshotIndex), []byte(`{"k":{"config.json":"`+dropHash+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	data := historyFileData{
		Events: []HistoryEvent{
			{ID: "a", SnapshotID: s2},
			{ID: "b", SnapshotID: s1},
		},
		Snapshots: map[string][]Entry{s1: {}, s2: {}},
	}
	if err := writeHistory(dir, data, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, s2)); err == nil {
		t.Fatal("dropped snapshot folder still present")
	}
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, "old-layout")); err == nil {
		t.Fatal("old layout folder still present")
	}
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, historyBlobsDir, dropHash)); err == nil {
		t.Fatal("unreferenced blob still present")
	}
	if _, err := os.Stat(filepath.Join(dir, historyFilesDir, historyBlobsDir, keepHash)); err != nil {
		t.Fatalf("kept blob missing: %v", err)
	}
}

func captureHistoryConfigs(dir, snapshotID string, entries []Entry) {
	idx, bodies := historyConfigIndex(dir, entries)
	storeHistoryConfigs(dir, snapshotID, idx, bodies)
}
