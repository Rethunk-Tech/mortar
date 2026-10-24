package support

import (
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	return fsx.WriteFile(filepath.Join(dataDir, startupProfileFlag), nil, 0o600)
}

// StartStartupProfile records the first 30 seconds of this run into the data folder when the previous run asked for
// it or MORTAR_PROFILE_STARTUP is set, replacing the last profile. It logs where the files went.
func StartStartupProfile(dataDir string) {
	if os.Getenv("MORTAR_PROFILE_STARTUP") == "" {
		if err := fsx.Remove(filepath.Join(dataDir, startupProfileFlag)); err != nil {
			return
		}
	}
	dir := filepath.Join(dataDir, StartupProfileDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("startup profile: %v", err)
		return
	}
	cpu, err := fsx.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		log.Printf("startup profile: %v", err)
		return
	}
	tr, err := fsx.Create(filepath.Join(dir, "trace.out"))
	if err != nil {
		_ = cpu.Close()
		log.Printf("startup profile: %v", err)
		return
	}
	if err := pprof.StartCPUProfile(cpu); err != nil {
		_, _ = cpu.Close(), tr.Close()
		log.Printf("startup profile: %v", err)
		return
	}
	if err := trace.Start(tr); err != nil {
		log.Printf("startup profile: trace: %v", err)
	}
	log.Printf("startup profile: recording %s into %s", startupProfileFor, dir)
	time.AfterFunc(startupProfileFor, func() {
		pprof.StopCPUProfile()
		trace.Stop()
		_, _ = cpu.Close(), tr.Close()
		log.Printf("startup profile: saved cpu.pprof and trace.out in %s", dir)
	})
}
