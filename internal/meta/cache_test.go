package meta

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEntryWithoutAValueReadsAsAbsent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	for name, body := range map[string]string{
		"releases": `{"fetched":"` + now + `","releases":[{"tag_name":"v0"}]}`,
		"version":  `{"fetched":"` + now + `","version":"4.7.0"}`,
		"bare":     `{"name":"x"}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if v, ok := Peek[string](&Client{CacheDir: dir}, name); ok {
			t.Errorf("%s read as present: %q", name, v)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "null"), []byte(`{"fetched":"`+now+`","value":null}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := Peek[[]string](&Client{CacheDir: dir}, "null"); !ok {
		t.Error(`a stored "value": null must read as present`)
	}
}

func TestRetiredCacheNames(t *testing.T) {
	t.Parallel()
	for rel, want := range map[string]bool{
		"github-o-r-0123456789ab-releases.json":    true,
		"github-o-r-v1-0123456789ab-notes.json":    true,
		"nexus/mod-stardewvalley-541.json":         true,
		"nexus-requirements-stardewvalley-1.json":  true,
		"github-o-r-0123456789ab-releases-v2.json": false,
		"github-o-r-v1-0123456789ab-notes-v2.json": false,
		"nexus/mod-v2-stardewvalley-541.json":      false,
		"smapi-release.json":                       false,
	} {
		if got := Retired(rel); got != want {
			t.Errorf("Retired(%q) = %v, want %v", rel, got, want)
		}
	}
}
