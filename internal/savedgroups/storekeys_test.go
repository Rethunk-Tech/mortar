package savedgroups

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestStoreKeysInListsEachGamesKeysOnce(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"stardew.json", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := StoreKeysIn(dir, func(string) ([]Mod, error) {
		return []Mod{{EntryKey: "a"}, {EntryKey: ""}, {EntryKey: "a"}, {EntryKey: "b"}}, nil
	})
	if err != nil || len(got) != 1 || !slices.Equal(got["stardew"], []string{"a", "b"}) {
		t.Fatalf("StoreKeysIn = %v, %v", got, err)
	}
}
