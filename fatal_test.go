package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	b, err := fsx.ReadFile(filepath.Join(dir, "crash.log"))
	if err != nil {
		t.Fatal(err)
	}
	if s := string(b); !strings.Contains(s, "first") || !strings.Contains(s, "second") || !strings.Contains(s, "goroutine 1") {
		t.Fatalf("crash.log = %q", s)
	}
}
