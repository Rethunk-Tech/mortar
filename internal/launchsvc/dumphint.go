package launchsvc

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/crashdump"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// dumpBlame is what the crash a run left behind says: the enabled mod whose DLL the process died in, or, when it was
// not a mod's, which part of the game was (reason). Windows dumps are looked for in the user's profile, which for a
// game under Proton is the prefix's; a Linux process is asked of systemd-coredump. Nothing found says nothing.
func (s *Service) dumpBlame(gameID string, run Run, mods []profile.Installed) (m profile.Installed, reason, evidence string, ok bool) {
	started, err := time.Parse(time.RFC3339Nano, run.Started)
	if err != nil {
		return profile.Installed{}, "", "", false
	}
	g, err := game.Require(gameID)
	if err != nil {
		return profile.Installed{}, "", "", false
	}
	exes := game.ProcessNames(g)
	for _, fault := range s.faultsSince(gameID, exes, started) {
		kind, mod := classifyFault(fault.Module, exes, mods)
		if kind == faultUnknown {
			continue
		}
		evidence = "crash dump: faulting module " + fault.Module
		if fault.Detail != "" {
			evidence += " (" + fault.Detail + ")"
		}
		switch kind {
		case faultMod:
			return mod, "", evidence, true
		case faultGame:
			reason = fmt.Sprintf("The game itself crashed (%s), not a mod's code.", fault.Module)
		case faultUnity:
			reason = fmt.Sprintf("Unity crashed (%s), not a mod's code.", fault.Module)
		case faultMono:
			reason = fmt.Sprintf("The Mono runtime crashed (%s), not a mod's code.", fault.Module)
		}
		return profile.Installed{}, reason, evidence, true
	}
	return profile.Installed{}, "", "", false
}

func (s *Service) faultsSince(gameID string, exes []string, since time.Time) []crashdump.Fault {
	if s.settings == nil {
		return nil
	}
	set := s.settings.Get()
	crashDumps, errA := game.PathIn(s.home, set, gameID, "", components.PathTemplate{Windows: `{localAppData}\CrashDumps`})
	temp, errB := game.PathIn(s.home, set, gameID, "", components.PathTemplate{Windows: `{localAppData}\Temp`})
	if errA == nil && errB == nil {
		return crashdump.Windows(crashDumps, temp, exes, since)
	}
	var out []crashdump.Fault
	for _, exe := range exes {
		comm := strings.TrimSuffix(exe, filepath.Ext(exe))
		if f, ok := crashdump.Coredump(context.Background(), comm[:min(len(comm), 15)], since); ok {
			out = append(out, f)
		}
	}
	return out
}

type faultKind int

const (
	faultUnknown faultKind = iota
	faultMod
	faultGame
	faultUnity
	faultMono
)

// classifyFault says whose code a faulting module is: an enabled mod's assembly (its EntryDll, or a DLL or shared
// library of that name in its folder), the game's own executable, Unity or Mono.
func classifyFault(module string, exes []string, mods []profile.Installed) (faultKind, profile.Installed) {
	name := fold(strings.TrimSuffix(module, filepath.Ext(module)))
	if name == "" {
		return faultUnknown, profile.Installed{}
	}
	for _, m := range mods {
		if m.Enabled && ownsModule(m, module) {
			return faultMod, m
		}
	}
	for _, exe := range exes {
		if fold(strings.TrimSuffix(exe, filepath.Ext(exe))) == name {
			return faultGame, profile.Installed{}
		}
	}
	switch {
	case strings.HasPrefix(name, "unityplayer") || name == "unity" || strings.HasPrefix(name, "libunity"):
		return faultUnity, profile.Installed{}
	case strings.HasPrefix(name, "mono") || strings.HasPrefix(name, "libmono"):
		return faultMono, profile.Installed{}
	}
	return faultUnknown, profile.Installed{}
}

// ownsModule is true when file is the mod's assembly: its EntryDll, or a file of that name at most three folders
// below the mod's folder.
func ownsModule(m profile.Installed, file string) bool {
	if m.EntryDll != "" && strings.EqualFold(filepath.Base(m.EntryDll), file) {
		return true
	}
	if m.Folder == "" {
		return false
	}
	root := filepath.Clean(m.Folder)
	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return fs.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if strings.Count(rel, string(os.PathSeparator)) >= 3 {
				return fs.SkipDir
			}
			return nil
		}
		found = strings.EqualFold(d.Name(), file)
		return nil
	})
	return found
}
