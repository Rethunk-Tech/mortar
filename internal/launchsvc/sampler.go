package launchsvc

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-gameproc"
	"github.com/Rethunk-Tech/go-nettrace"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	startupSampleLimit = 15 * time.Minute
	livenessEvery      = time.Second
)

type startupSamples struct {
	Schema        int              `json:"schema"`
	Mods          map[string]int64 `json:"mods"`
	OtherMs       int64            `json:"otherMs"`
	ThreadSamples int              `json:"threadSamples"`
}

func (s *Service) sampleStartup(parent context.Context, g game.Game, profileID, modsDir string, before map[string]bool, stopped func()) {
	defer stopped()
	// A test launch stops the game the moment the report appears and cancels parent with it; the sampler must still
	// see that report, so it ends on its own limit or when the game process is gone.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), startupSampleLimit)
	defer cancel()

	process, err := s.waitForSampleProcess(ctx, g, profileID, modsDir)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	profileDir, err := s.profiles.ProfileDir(g.ID(), profileID)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	session, file, err := startSampler(ctx, process.PID)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	detached := context.WithoutCancel(ctx)
	reportID, reportFound := s.waitForSampleReport(ctx, g, profileID, modsDir, filepath.Join(profileDir, startupDir), before)
	stopCtx, stopCancel := context.WithTimeout(detached, 30*time.Second)
	stopErr := session.Stop(stopCtx)
	stopCancel()
	stopped()
	defer func() {
		_ = file.Close()
		_ = safefs.Remove(file.Name())
	}()
	if stopErr != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), stopErr)
		return
	}
	if !reportFound {
		return
	}
	samples, err := writeStartupSamples(file, filepath.Join(profileDir, startupDir), reportID, modsDir)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	log.Printf("startup sampler: %s: %d main-thread samples, %d mods charged, %d ms other", g.ID(), samples.ThreadSamples, len(samples.Mods), samples.OtherMs)
}

// startSampler traces the game's startup into a temp file the caller closes and removes once the session has stopped.
func startSampler(ctx context.Context, pid int) (*nettrace.Session, *os.File, error) {
	file, err := os.CreateTemp("", "mortar-startup-*.nettrace")
	if err != nil {
		return nil, nil, err
	}
	for {
		session, err := nettrace.StartTracing(ctx, pid, nettrace.Config{}, file)
		if err == nil {
			return session, file, nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			err = ctx.Err()
		case <-timer.C:
			if errors.Is(err, nettrace.ErrNoEndpoint) {
				continue
			}
		}
		_ = file.Close()
		_ = safefs.Remove(file.Name())
		return nil, nil, err
	}
}

func (s *Service) waitForSampleProcess(ctx context.Context, g game.Game, profileID, modsDir string) (gameproc.Process, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		processes, err := s.procsFor(g, modsDir, profileID)
		if err == nil && len(processes) != 0 {
			return processes[0], nil
		}
		select {
		case <-ctx.Done():
			return gameproc.Process{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) waitForSampleReport(ctx context.Context, g game.Game, profileID, modsDir, startupPath string, before map[string]bool) (string, bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var scanned time.Time
	for {
		if id := newStartupReport(startupPath, before); id != "" {
			return id, true
		}
		// Finding the game process scans all of /proc, which would compete with the startup being timed.
		if now := time.Now(); now.Sub(scanned) >= livenessEvery {
			scanned = now
			processes, err := s.procsFor(g, modsDir, profileID)
			if err == nil && len(processes) == 0 {
				return "", false
			}
		}
		select {
		case <-ctx.Done():
			return "", false
		case <-ticker.C:
		}
	}
}

func startupReportIDs(path string) (map[string]bool, error) {
	names, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool)
	for _, name := range names {
		if strings.HasSuffix(name, ".samples.json") {
			continue
		}
		ids[strings.TrimSuffix(filepath.Base(name), ".json")] = true
	}
	return ids, nil
}

func newStartupReport(path string, before map[string]bool) string {
	names, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		return ""
	}
	for _, name := range names {
		if strings.HasSuffix(name, ".samples.json") {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(name), ".json")
		if !before[id] {
			return id
		}
	}
	return ""
}

func writeStartupSamples(file *os.File, startupPath, reportID, modsDir string) (startupSamples, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return startupSamples{}, err
	}
	trace, err := nettrace.Parse(file)
	if err != nil {
		return startupSamples{}, err
	}
	assemblyToMod, err := assembliesToMods(modsDir)
	if err != nil {
		return startupSamples{}, err
	}
	threadID, ok := mainSampleThread(trace.Samples, assemblyToMod)
	if !ok {
		return startupSamples{}, errors.New("no mod frames were sampled")
	}
	samples := charge(trace.Samples, threadID, assemblyToMod)
	return samples, datadir.WriteJSON(filepath.Join(startupPath, reportID+".samples.json"), samples)
}

func assembliesToMods(modsDir string) (map[string]string, error) {
	assemblyToMod := make(map[string]string)
	err := filepath.WalkDir(modsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(entry.Name(), smapimanifest.FileName) {
			return nil
		}
		data, err := safefs.ReadFile(path)
		if err != nil {
			return err
		}
		// SMAPI skips a mod whose manifest it cannot read, so such a mod has no frames to charge.
		if parsed, parseErr := smapimanifest.Parse(data); parseErr == nil {
			return mapModAssemblies(assemblyToMod, filepath.Dir(path), parsed.ModID())
		}
		return nil
	})
	return assemblyToMod, err
}

func mapModAssemblies(assemblyToMod map[string]string, dir string, uniqueID mod.ID) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, one := range entries {
		if one.IsDir() || !strings.EqualFold(filepath.Ext(one.Name()), ".dll") {
			continue
		}
		assemblyToMod[strings.TrimSuffix(one.Name(), filepath.Ext(one.Name()))] = uniqueID.Local()
	}
	return nil
}

func mainSampleThread(samples []nettrace.Sample, assemblyToMod map[string]string) (uint64, bool) {
	counts := make(map[uint64]int)
	for _, sample := range samples {
		for _, frame := range sample.Frames {
			if modOf(frame.Assembly, assemblyToMod) != "" {
				counts[sample.ThreadID]++
				break
			}
		}
	}
	var threadID uint64
	count := 0
	for candidate, value := range counts {
		if value > count {
			threadID, count = candidate, value
		}
	}
	return threadID, count != 0
}
