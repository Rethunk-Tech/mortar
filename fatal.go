package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// shownError is a startup failure the user has already seen in a dialog, so main does not show it again.
type shownError struct{ error }

func (e shownError) Unwrap() error { return e.error }

// startupFailureText is what the dialog says when Mortar cannot start: a windowsgui build has no console, so without
// it a failed start looks like nothing happened.
func startupFailureText(err error) string {
	return i18n.T("Mortar could not start:\n\n{error}\n\nThe details are in mortar.log in Mortar's data folder.", "error", err.Error())
}

// logFolder is the data folder holding mortar.log and crash.log, "" when it cannot be found (the failure may be that).
func logFolder() string {
	if dir, err := datadir.Dir(); err == nil {
		return dir
	}
	return ""
}

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
		showStartupError(i18n.T("Mortar stopped because of an error:\n\n{error}\n\nThe details are in crash.log in {path}.", "error", fmt.Sprint(d.Error), "path", dataDir), dataDir)
		os.Exit(2)
	}
}

func appendCrash(dataDir string, d *application.PanicDetails) error {
	f, err := safefs.OpenFile(filepath.Join(dataDir, "crash.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, werr := fmt.Fprintf(f, "%s panic: %v\n%s\n", d.Time.Format("2006-01-02T15:04:05Z07:00"), d.Error, d.FullStackTrace)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}
