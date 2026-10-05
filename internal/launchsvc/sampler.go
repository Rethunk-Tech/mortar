package launchsvc

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/sampler"
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
	session, err := startSampler(ctx, process.PID)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	detached := context.WithoutCancel(ctx)
	profileDir, err := s.profiles.ProfileDir(g.ID(), profileID)
	if err != nil {
		_, _ = session.Stop(detached)
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	reportID, reportFound := s.waitForSampleReport(ctx, g, profileID, modsDir, filepath.Join(profileDir, startupDir), before)
	stopCtx, stopCancel := context.WithTimeout(detached, 30*time.Second)
	tracePath, stopErr := session.Stop(stopCtx)
	stopCancel()
	stopped()
	if stopErr != nil {
		_ = os.Remove(tracePath)
		log.Printf("startup sampler: %s: %v", g.ID(), stopErr)
		return
	}
	defer func() { _ = os.Remove(tracePath) }()
	if !reportFound {
		return
	}
	samples, err := writeStartupSamples(tracePath, filepath.Join(profileDir, startupDir), reportID, modsDir)
	if err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	log.Printf("startup sampler: %s: %d main-thread samples, %d mods charged, %d ms other", g.ID(), samples.ThreadSamples, len(samples.Mods), samples.OtherMs)
}

func startSampler(ctx context.Context, pid int) (*sampler.Session, error) {
	var last error
	for {
		session, err := sampler.Start(ctx, pid)
		if err == nil {
			return session, nil
		}
		last = err
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, last
		case <-timer.C:
		}
	}
}

func (s *Service) waitForSampleProcess(ctx context.Context, g game.Game, profileID, modsDir string) (launch.Process, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		processes, err := s.procsFor(g, modsDir, profileID)
		if err == nil && len(processes) != 0 {
			return processes[0], nil
		}
		select {
		case <-ctx.Done():
			return launch.Process{}, ctx.Err()
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

func writeStartupSamples(tracePath, startupPath, reportID, modsDir string) (startupSamples, error) {
	trace, err := sampler.ParseFile(tracePath)
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
	charged := sampler.Charge(trace.Samples, threadID, assemblyToMod)
	samples := startupSamples{
		Schema:        1,
		Mods:          charged.Mods,
		OtherMs:       charged.OtherMs,
		ThreadSamples: charged.ThreadSamples,
	}
	return samples, datadir.WriteJSON(filepath.Join(startupPath, reportID+".samples.json"), samples)
}

func assembliesToMods(modsDir string) (map[string]string, error) {
	assemblyToMod := make(map[string]string)
	err := filepath.WalkDir(modsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(entry.Name(), manifest.FileName) {
			return nil
		}
		data, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		// SMAPI skips a mod whose manifest it cannot read, so such a mod has no frames to charge.
		if parsed, parseErr := manifest.Parse(data); parseErr == nil {
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

func mainSampleThread(samples []sampler.Sample, assemblyToMod map[string]string) (uint32, bool) {
	counts := make(map[uint32]int)
	for _, sample := range samples {
		for _, frame := range sample.Frames {
			if mappedAssembly(frame.Assembly, assemblyToMod) {
				counts[sample.ThreadID]++
				break
			}
		}
	}
	var threadID uint32
	count := 0
	for candidate, value := range counts {
		if value > count {
			threadID, count = candidate, value
		}
	}
	return threadID, count != 0
}

func mappedAssembly(assembly string, assemblyToMod map[string]string) bool {
	assembly = strings.TrimSpace(strings.SplitN(assembly, ",", 2)[0])
	assembly = strings.TrimSuffix(filepath.Base(strings.ReplaceAll(assembly, `\`, `/`)), ".dll")
	_, ok := assemblyToMod[assembly]
	return ok
}
