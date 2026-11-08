package launchsvc

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// sandboxSpec is what a sandboxed launch of profileID may write besides the install and its Proton prefix, which
// game.Starter adds: the profile, the saves the launch plays, the folders mods are placed in, and the shader caches.
func (s *Service) sandboxSpec(gameID, profileID string, inst game.Install, plan *launchplan.Plan, extra string) (*sandbox.Spec, error) {
	spec := &sandbox.Spec{Home: s.home, Runtime: os.Getenv("XDG_RUNTIME_DIR")}
	if spec.Runtime != "" {
		spec.Sockets = []string{os.Getenv("XAUTHORITY"), filepath.Join(spec.Runtime, "pipewire-0"), filepath.Join(spec.Runtime, "pulse")}
		if wl := os.Getenv("WAYLAND_DISPLAY"); wl != "" {
			spec.Sockets = append(spec.Sockets, filepath.Join(spec.Runtime, wl))
		}
	}
	folders, err := sandbox.ExtraFolders(extra, s.home)
	if err != nil {
		return nil, usererr.Wrap(usererr.Invalid, err)
	}
	spec.RW = append(spec.RW, folders...)
	roots, err := s.planRoots(gameID, inst, plan)
	if err != nil {
		return nil, err
	}
	spec.RW = slices.AppendSeq(spec.RW, maps.Values(roots))
	if profileID != "" && s.profiles != nil {
		dir, err := s.profiles.ProfileDir(gameID, profileID)
		if err != nil {
			return nil, err
		}
		spec.RW = append(spec.RW, dir)
		spec.CacheDir = filepath.Join(dir, "sandbox-cache")
	}
	// Everything the game writes beside its saves (SMAPI's ErrorLogs, startup preferences, screenshots) shares their
	// folder; binding only the saves would send those to the throwaway home.
	if inst.Platform == "linux" {
		dirs := game.DataDirs(s.home, s.settings.Get(), gameID, inst.ID)
		spec.RW = append(spec.RW, dirs...)
		if len(dirs) == 0 {
			spec.Warn = "this game's save folder is not known, so anything it writes outside the install is lost when it exits; add the folder under Extra folders"
		}
	}
	if game.HasSaves(gameID) {
		saves, err := s.playedSavesDir(s.settings.Get(), gameID, profileID, inst.ID)
		if err != nil {
			return nil, err
		}
		spec.RW = append(spec.RW, saves)
	}
	// Creating a folder inside a Proton prefix before Proton has made it would leave a half-built prefix, and the
	// prefix is bound whole anyway; a native game's folders are made here so its first save lands in the bound one.
	var rw []string
	for _, dir := range spec.RW {
		if inst.Platform == "linux" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, err
			}
		} else if _, err := os.Stat(dir); err != nil {
			continue
		}
		rw = append(rw, dir)
	}
	if spec.CacheDir != "" {
		if err := os.MkdirAll(spec.CacheDir, 0o750); err != nil {
			return nil, err
		}
	}
	spec.RW = rw
	return spec, nil
}

// SandboxReport is what a stored run says about the sandbox.
type SandboxReport struct {
	// Writable is the folders the run could write; empty when the run was not sandboxed.
	Writable []string `json:"writable"`
	// Denied are the paths its logs show it reached for outside them.
	Denied []SandboxDenial `json:"denied"`
	// Extra is the profile's current list of extra folders, which Allow adds to.
	Extra string `json:"extra"`
}

// SandboxDenial is a refused path and the folder to offer allowing for it; empty when none is safe to offer.
type SandboxDenial struct {
	Path   string `json:"path"`
	Folder string `json:"folder"`
}

// SandboxReport reads run runID (the newest when empty) of a profile.
func (s *Service) SandboxReport(gameID, profileID, runID string) (SandboxReport, error) {
	report := SandboxReport{Writable: []string{}, Denied: []SandboxDenial{}}
	if g := game.Find(gameID); g != nil {
		scope := settings.Scope{Game: gameID, Install: installOf(g), Profile: profileID}
		report.Extra = settings.ResolveAt(s.settings.Get(), "sandboxExtraFolders", scope, launchOverrides(s.profiles, gameID, profileID))
	}
	runs, err := s.Runs(gameID, profileID)
	if err != nil {
		return report, err
	}
	for _, r := range runs {
		if runID != "" && r.ID != runID {
			continue
		}
		if len(r.Sandbox) == 0 {
			return report, nil
		}
		text, player, err := s.storedRunLogs(gameID, profileID, r.ID)
		if err != nil {
			return report, err
		}
		report.Writable = r.Sandbox
		for _, path := range sandbox.Denials([]string{text, player}, r.Sandbox, s.home) {
			report.Denied = append(report.Denied, SandboxDenial{Path: path, Folder: sandbox.Allowable(path, s.home)})
		}
		return report, nil
	}
	return report, nil
}
