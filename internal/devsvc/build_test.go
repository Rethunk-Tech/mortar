package devsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWatchedSourcesCoverCompatAndAssets(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	// Build on save is off: the sources are still watched, so the card can say they changed.
	dir := fixtureProject(t, svc)
	got := WatchedSources(svc.Settings.Get(), []string{"lethal-company"})["lethal-company"]
	src := filepath.Join(dir, "src", "ShopDrop")
	for _, want := range []string{src, filepath.Join(src, "Compat"), filepath.Join(dir, "assets")} {
		if !slices.Contains(got, want) {
			t.Errorf("%s is not watched: %v", want, got)
		}
	}
}

func TestBuildRemovesBundlesAssetsNoLongerHolds(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	out := filepath.Join(dir, "artifacts", "bin", "ShopDrop", "debug")
	for path, body := range map[string]string{
		filepath.Join(dir, "assets", "kept.bundle"): "a",
		filepath.Join(out, "kept.bundle"):           "a",
		filepath.Join(out, "gone.bundle"):           "b",
		filepath.Join(out, "ShopDrop.dll"):          "c",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc.Run = func(context.Context, string, string, ...string) ([]byte, error) { return nil, nil }
	if _, err := svc.Build(t.Context(), "lethal-company", dir); err != nil {
		t.Fatal(err)
	}
	var left []string
	entries, _ := os.ReadDir(out)
	for _, e := range entries {
		left = append(left, e.Name())
	}
	if want := []string{"ShopDrop.dll", "kept.bundle"}; !slices.Equal(left, want) {
		t.Errorf("output holds %v, want %v", left, want)
	}
}
