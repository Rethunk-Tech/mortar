package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// logAppError sends Wails runtime errors to Mortar's log; without a handler they only reach a logger that discards
// its output in release builds.
func logAppError(err error) {
	slog.Error("wails", "err", err)
}

// panicHandler returns the Wails PanicHandler: it records the stack in crash.log, which the next start reports,
// then exits, as Wails' own default does after printing to a logger nobody reads.
func panicHandler(dataDir string) func(*application.PanicDetails) {
	return func(d *application.PanicDetails) {
		slog.Error("panic", "err", d.Error)
		_ = appendCrash(dataDir, d)
		os.Exit(2)
	}
}

func appendCrash(dataDir string, d *application.PanicDetails) error {
	f, err := fsx.OpenFile(filepath.Join(dataDir, "crash.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, werr := fmt.Fprintf(f, "%s panic: %v\n%s\n", d.Time.Format("2006-01-02T15:04:05Z07:00"), d.Error, d.FullStackTrace)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}
