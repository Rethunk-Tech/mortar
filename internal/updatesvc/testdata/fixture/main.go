// Command fixture is a server-mode Wails app that drives the real updatesvc.Service, so the end-to-end updater test
// can swap its own binary on disk. Build: go build -tags "server production updatetest" -ldflags "-X main.version=..."
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Rethunk-AI/mortar/internal/updatesvc"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var version = "0.0.0"

func main() {
	app := application.New(application.Options{
		Name:   "fixture",
		Assets: application.AssetOptions{Handler: http.NotFoundHandler()},
	})
	svc := &updatesvc.Service{}
	key, err := base64.StdEncoding.DecodeString(os.Getenv("FIXTURE_PUBKEY_B64"))
	if err != nil {
		fatal(err)
	}
	if err := updatesvc.Configure(svc, app.Updater, version, key, "", nil, ""); err != nil {
		fatal(err)
	}
	record("start version=%s", version)
	go func() {
		defer app.Quit()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		rel, err := svc.Check(ctx)
		if err != nil || rel == nil {
			record("check release=%v err=%v", rel, err)
			return
		}
		record("check found=%s", rel.Version)
		if err := svc.Install(ctx); err != nil {
			record("install err=%v", err)
			return
		}
		record("staged")
		// Restart quits the app, so the line has to be written first.
		record("restarting")
		if err := svc.Restart(ctx); err != nil {
			record("restart err=%v", err)
			return
		}
		select {}
	}()
	if err := app.Run(); err != nil {
		fatal(err)
	}
}

// record appends one line to FIXTURE_LOG, the channel the test reads results from.
func record(format string, args ...any) {
	f, err := os.OpenFile(os.Getenv("FIXTURE_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, format+"\n", args...)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
