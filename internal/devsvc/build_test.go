package devsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

func TestWatchedSourcesIncludeSubfoldersButNotBuildOutput(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	src := filepath.Join(dir, "src", "ShopDrop")
	for _, d := range []string{"Patches", "obj/x"} {
		if err := os.MkdirAll(filepath.Join(src, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	got := WatchedSources(svc.Settings.Get(), []string{"lethal-company"})["lethal-company"]
	if !slices.Contains(got, filepath.Join(src, "Patches")) || slices.Contains(got, filepath.Join(src, "obj")) || slices.Contains(got, filepath.Join(src, "obj", "x")) {
		t.Errorf("watched %v", got)
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

func TestBuildKeepsItsOutputForTheDiagnosticsBundle(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.BuildLogDir = t.TempDir()
	dir := fixtureProject(t, svc)
	svc.Run = func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte("Plugin.cs(3,1): error CS1002: ; expected [ShopDrop.csproj]\n"), errors.New("exit status 1")
	}
	if r, err := svc.Build(t.Context(), "lethal-company", dir); err != nil || r.OK {
		t.Fatalf("build = %+v, %v", r, err)
	}
	logs, _ := filepath.Glob(filepath.Join(svc.BuildLogDir, "*.log"))
	if len(logs) != 1 {
		t.Fatalf("kept logs %v", logs)
	}
	body, _ := os.ReadFile(logs[0])
	for _, want := range []string{"dotnet build", "result: failed: exit status 1", "error CS1002"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("kept log lacks %q:\n%s", want, body)
		}
	}
}

// queuedBehindTrash runs call while the project's lock is held as Trash holds it, then removes the folder and forgets
// the project before releasing the lock. It returns what ran and call's error.
func queuedBehindTrash(t *testing.T, call func(svc *Service, dir string) error) ([]string, error) {
	t.Helper()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	var ran []string
	svc.Run = func(_ context.Context, wd, name string, _ ...string) ([]byte, error) {
		ran = append(ran, name+" in "+wd)
		return nil, nil
	}
	release := holdBuild(dir)
	done := make(chan error, 1)
	go func() { done <- call(svc, dir) }()
	// Nothing signals that the call has read the project and queued on the lock, so give it time to; a call that is
	// late only reaches the project lookup after Forget and passes without exercising the lock.
	time.Sleep(100 * time.Millisecond)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := svc.Forget("lethal-company", dir); err != nil {
		t.Fatal(err)
	}
	release()
	return ran, <-done
}

func requireGone(t *testing.T, ran []string, err error) {
	t.Helper()
	if len(ran) != 0 {
		t.Errorf("ran %v on a removed project", ran)
	}
	var ue *usererr.Error
	if !errors.As(err, &ue) || ue.Kind() != usererr.NotFound {
		t.Errorf("error = %v, want NotFound", err)
	}
}

func TestBuildQueuedBehindTrashDoesNotRunOnTheRemovedProject(t *testing.T) {
	ran, err := queuedBehindTrash(t, func(svc *Service, dir string) error {
		_, err := svc.Build(t.Context(), "lethal-company", dir)
		return err
	})
	requireGone(t, ran, err)
}

func TestPackQueuedBehindTrashDoesNotRunOnTheRemovedProject(t *testing.T) {
	ran, err := queuedBehindTrash(t, func(svc *Service, dir string) error {
		_, err := svc.Pack(t.Context(), "lethal-company", dir)
		return err
	})
	requireGone(t, ran, err)
}
