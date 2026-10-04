package launchsvc

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/sampler"
)

const startupSampleLimit = 15 * time.Minute

type startupSamples struct {
	Schema        int              `json:"schema"`
	Mods          map[string]int64 `json:"mods"`
	OtherMs       int64            `json:"otherMs"`
	ThreadSamples int              `json:"threadSamples"`
}

type startupManifest struct {
	UniqueID string `json:"UniqueID"`
	EntryDLL string `json:"EntryDll"`
}

func (s *Service) sampleStartup(parent context.Context, g game.Game, profileID, modsDir string, before map[string]bool) {
	ctx, cancel := context.WithTimeout(parent, startupSampleLimit)
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
	profileDir, err := s.profiles.ProfileDir(g.ID(), profileID)
	if err != nil {
		_, _ = session.Stop(context.Background())
		log.Printf("startup sampler: %s: %v", g.ID(), err)
		return
	}
	reportID, reportFound := s.waitForSampleReport(ctx, g, profileID, modsDir, filepath.Join(profileDir, startupDir), before)
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	tracePath, stopErr := session.Stop(stopCtx)
	stopCancel()
	if stopErr != nil {
		_ = os.Remove(tracePath)
		log.Printf("startup sampler: %s: %v", g.ID(), stopErr)
		return
	}
	defer os.Remove(tracePath)
	if !reportFound {
		return
	}
	if err := writeStartupSamples(tracePath, filepath.Join(profileDir, startupDir), reportID, modsDir); err != nil {
		log.Printf("startup sampler: %s: %v", g.ID(), err)
	}
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
			if last != nil {
				return nil, last
			}
			return nil, ctx.Err()
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
	for {
		if id := newStartupReport(startupPath, before); id != "" {
			return id, true
		}
		processes, err := s.procsFor(g, modsDir, profileID)
		if err == nil && len(processes) == 0 {
			return "", false
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

func writeStartupSamples(tracePath, startupPath, reportID, modsDir string) error {
	trace, err := sampler.ParseFile(tracePath)
	if err != nil {
		return err
	}
	assemblyToMod, err := assembliesToMods(modsDir)
	if err != nil {
		return err
	}
	threadID, ok := mainSampleThread(trace.Samples, assemblyToMod)
	if !ok {
		return errors.New("no mod frames were sampled")
	}
	charged := sampler.Charge(trace.Samples, threadID, assemblyToMod)
	samples := startupSamples{
		Schema:        1,
		Mods:          charged.Mods,
		OtherMs:       charged.OtherMs,
		ThreadSamples: charged.ThreadSamples,
	}
	return datadir.WriteJSON(filepath.Join(startupPath, reportID+".samples.json"), samples)
}

func assembliesToMods(modsDir string) (map[string]string, error) {
	assemblyToMod := make(map[string]string)
	err := filepath.WalkDir(modsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(entry.Name(), "manifest.json") {
			return nil
		}
		var manifest startupManifest
		found, err := datadir.ReadJSON(path, &manifest)
		if err != nil {
			return err
		}
		if !found || manifest.UniqueID == "" {
			return nil
		}
		dir := filepath.Dir(path)
		if manifest.EntryDLL != "" {
			assemblyToMod[strings.TrimSuffix(filepath.Base(manifest.EntryDLL), filepath.Ext(manifest.EntryDLL))] = manifest.UniqueID
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, one := range entries {
			if one.IsDir() || !strings.EqualFold(filepath.Ext(one.Name()), ".dll") {
				continue
			}
			assemblyToMod[strings.TrimSuffix(one.Name(), filepath.Ext(one.Name()))] = manifest.UniqueID
		}
		return nil
	})
	return assemblyToMod, err
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
