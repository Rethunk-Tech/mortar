package support

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

const (
	// startupProfileFlag in the data folder asks the next start to record a profile; that start removes it.
	startupProfileFlag = "profile-next-start"
	// StartupProfileDir holds the newest startup profile: cpu.pprof (where the CPU went) and trace.out (what each
	// goroutine waited on, which a CPU profile cannot show).
	StartupProfileDir = "startup-profile"
	startupProfileFor = 30 * time.Second
)

// RequestStartupProfile makes the next start of Mortar record a startup profile.
func RequestStartupProfile(dataDir string) error {
	return safefs.WriteFile(filepath.Join(dataDir, startupProfileFlag), nil, 0o600)
}

// StartStartupProfile records the first 30 seconds of this run into the data folder when the previous run asked for
// it, replacing the last profile. It logs where the files went.
func StartStartupProfile(dataDir string) {
	if err := safefs.Remove(filepath.Join(dataDir, startupProfileFlag)); err != nil {
		return
	}
	dir := filepath.Join(dataDir, StartupProfileDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		slog.Warn("startup profile", "err", err)
		return
	}
	cpu, err := safefs.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		slog.Warn("startup profile", "err", err)
		return
	}
	tr, err := safefs.Create(filepath.Join(dir, "trace.out"))
	if err != nil {
		_ = cpu.Close()
		slog.Warn("startup profile", "err", err)
		return
	}
	if err := pprof.StartCPUProfile(cpu); err != nil {
		_, _ = cpu.Close(), tr.Close()
		slog.Warn("startup profile", "err", err)
		return
	}
	if err := trace.Start(tr); err != nil {
		slog.Warn("startup profile: trace", "err", err)
	}
	slog.Info("startup profile: recording", "what", startupProfileFor, "dir", dir)
	time.AfterFunc(startupProfileFor, func() {
		pprof.StopCPUProfile()
		trace.Stop()
		_, _ = cpu.Close(), tr.Close()
		slog.Info("startup profile: saved cpu.pprof and trace.out", "dir", dir)
	})
}
