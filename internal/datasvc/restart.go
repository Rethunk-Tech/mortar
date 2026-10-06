package datasvc

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
)

// RestartSelf starts this executable again, without the old arguments (a fresh start is what a data move needs, and
// they may name a link already handled), and exits.
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Before the new process starts, which reads the record when it sets the log aside.
	logCleanShutdown()
	cmd := exec.CommandContext(context.Background(), selfexe.Launchable(exe))
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

// logCleanShutdown ends the data folder's mortar.log with the clean-shutdown record the next start reads to tell a
// quit from a crash. Exiting here skips the app's own shutdown, and after a move the log the next start reads is the
// copy in the new folder, not the file this run has been writing.
func logCleanShutdown() {
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	f, err := fsx.OpenFile(filepath.Join(dir, "mortar.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	slog.New(slog.NewTextHandler(f, nil)).Info("shutdown", "clean", true)
	_ = f.Close()
}
