package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// An installed copy answers the app id, whose D-Bus name nxm and share handoffs arrive on; only a portable copy,
// which must not collide with it, has an id of its own.
func TestSingleInstanceIDIsTheAppIDForTheInstalledApp(t *testing.T) {
	testfs.DataHome(t)
	if got := singleInstanceID(filepath.Join(t.TempDir(), "somewhere")); got != appversion.AppID {
		t.Fatalf("singleInstanceID = %q, want the app id %q", got, appversion.AppID)
	}
}

// logFolder names the data folder holding mortar.log, and "" when that folder cannot be found (the failure may be
// that), so the dialog never points at a folder that is not there.
func TestLogFolderIsTheDataFolderOrEmpty(t *testing.T) {
	home := testfs.DataHome(t)
	dir := logFolder()
	if dir == "" {
		t.Fatalf("logFolder is empty with a data home at %s", home)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("logFolder %q: %v", dir, err)
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("HOME", "")
	if got := logFolder(); got != "" {
		t.Errorf("logFolder with no home = %q, want empty", got)
	}
}

// A start that cannot find its data folder still records why in the default location's startup.log, since the log
// inside the data folder is the thing that is missing.
func TestLogStartupFailureWritesTheDefaultStartupLog(t *testing.T) {
	testfs.DataHome(t)
	def, err := datadir.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	logStartupFailure(errors.New("the drive is unplugged"))
	got := testfs.Read(t, filepath.Join(def, "startup.log"))
	if !strings.Contains(got, "the drive is unplugged") {
		t.Fatalf("startup.log = %q", got)
	}
}

// A window geometry round-trips through the data folder, and one written for a monitor that is no longer there comes
// back on the screen that is, so the window never opens offscreen.
func TestWindowGeometryRoundTripsAndStaysOnScreen(t *testing.T) {
	dir := t.TempDir()
	screens := []screenRect{{X: 0, Y: 0, Width: 1920, Height: 1080}}
	saveWindowGeom(dir, windowGeom{X: 200, Y: 120, W: 1280, H: 720})
	if got, ok := loadWindowGeom(dir, screens); !ok || got != (windowGeom{X: 200, Y: 120, W: 1280, H: 720}) {
		t.Fatalf("round trip = %+v, %v", got, ok)
	}
	saveWindowGeom(dir, windowGeom{X: -3000, Y: -2000, W: 1280, H: 720})
	if got, ok := loadWindowGeom(dir, screens); !ok || got.X != 0 || got.Y != 0 {
		t.Fatalf("offscreen geometry = %+v, %v", got, ok)
	}
	if err := os.WriteFile(filepath.Join(dir, "window.json"), []byte(`{"w":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadWindowGeom(dir, screens); ok {
		t.Error("a geometry with no size was used")
	}
}

// Without a Wails app there are no screens to clamp to, and the saved geometry is used as it is.
func TestScreensOfNoAppIsEmpty(t *testing.T) {
	if got := screensOf(nil); got != nil {
		t.Fatalf("screensOf(nil) = %+v", got)
	}
}

// A quit question asked while the window was closed is handed to the next page once, then forgotten.
func TestPendingQuitIsHandedOverOnce(t *testing.T) {
	s := &QuitService{pending: "Downloads will be interrupted"}
	if got := s.PendingQuit(); got == "" {
		t.Fatal("the waiting question was not handed over")
	}
	if got := s.PendingQuit(); got != "" {
		t.Fatalf("the question was handed over twice: %q", got)
	}
}

// A pid running another program is never ended: after Mortar exits the system may hand its number to anything.
func TestRunsMortarChecksTheImageName(t *testing.T) {
	if !runsMortar(os.Getpid()) {
		t.Error("this test's own process is not recognised as Mortar's image")
	}
	if runsMortar(-1) {
		t.Error("a pid with no process was taken for Mortar")
	}
	if runsMortar(1) {
		t.Error("pid 1, running another program, was taken for Mortar")
	}
}
