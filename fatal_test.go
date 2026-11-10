package main

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestAppendCrashKeepsEarlierStacks(t *testing.T) {
	dir := t.TempDir()
	for _, msg := range []string{"first", "second"} {
		d := &application.PanicDetails{Error: errors.New(msg), Time: time.Now(), FullStackTrace: "goroutine 1 [running]:"}
		if err := appendCrash(dir, d); err != nil {
			t.Fatal(err)
		}
	}
	b, err := safefs.ReadFile(filepath.Join(dir, "crash.log"))
	if err != nil {
		t.Fatal(err)
	}
	if s := string(b); !strings.Contains(s, "first") || !strings.Contains(s, "second") || !strings.Contains(s, "goroutine 1") {
		t.Fatalf("crash.log = %q", s)
	}
	if emptied, crashed := checkCrashLogAt(dir, maxCrashBytes); !crashed || emptied {
		t.Fatalf("the next start: emptied %v, crashed %v; want the trace kept and reported", emptied, crashed)
	}
	if checkCrashLog(dir) {
		t.Fatal("a small crash.log was emptied")
	}
	if err := appendCrash(filepath.Join(dir, "no-such-folder"), &application.PanicDetails{Error: errors.New("x")}); err == nil {
		t.Error("a folder that is not there accepted the crash")
	}
}

// A failure the user has already seen in a dialog unwraps to its cause, and the dialog names the error and where the
// log is, since a windowsgui build has no console to print to.
func TestStartupFailureTextNamesTheErrorAndTheLog(t *testing.T) {
	cause := errors.New("the drive is unplugged")
	wrapped := shownError{cause}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("a shown error does not unwrap to its cause: %v", wrapped)
	}
	text := startupFailureText(wrapped)
	if !strings.Contains(text, cause.Error()) || !strings.Contains(text, "mortar.log") {
		t.Fatalf("startupFailureText = %q", text)
	}
}

// A Wails runtime error reaches Mortar's log, and a startup step records its phase and the time since the last one.
func TestLogAppErrorAndStartupStepWriteTheLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	f, err := safefs.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	logAppError(errors.New("the page is gone"))
	startupStep("catalog")
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	got := testfs.Read(t, path)
	for _, want := range []string{"the page is gone", "wails", "catalog", "sinceStart"} {
		if !strings.Contains(got, want) {
			t.Errorf("log lacks %q: %q", want, got)
		}
	}
	start := time.Now()
	if since(start) < 0 {
		t.Error("since reported a negative duration")
	}
}
