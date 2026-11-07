package launchsvc

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// sandboxSpec is what a sandboxed launch of profileID may write besides the install and its Proton prefix, which
// game.Starter adds: the profile, the saves the launch plays, the folders mods are placed in, and the shader caches.
func (s *Service) sandboxSpec(gameID, profileID string, inst game.Install, plan *launchplan.Plan) (*sandbox.Spec, error) {
	spec := &sandbox.Spec{Home: s.home, Runtime: os.Getenv("XDG_RUNTIME_DIR")}
	if spec.Runtime != "" {
		spec.Sockets = []string{os.Getenv("XAUTHORITY"), filepath.Join(spec.Runtime, "pipewire-0"), filepath.Join(spec.Runtime, "pulse")}
		if wl := os.Getenv("WAYLAND_DISPLAY"); wl != "" {
			spec.Sockets = append(spec.Sockets, filepath.Join(spec.Runtime, wl))
		}
	}
	roots, err := s.planRoots(gameID, inst, plan)
	if err != nil {
		return nil, err
	}
	for _, dir := range roots {
		spec.RW = append(spec.RW, dir)
	}
	if profileID != "" && s.profiles != nil {
		dir, err := s.profiles.ProfileDir(gameID, profileID)
		if err != nil {
			return nil, err
		}
		spec.RW = append(spec.RW, dir)
		spec.CacheDir = filepath.Join(dir, "sandbox-cache")
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
	Denied []string `json:"denied"`
}

// SandboxReport reads run runID (the newest when empty) of a profile.
func (s *Service) SandboxReport(gameID, profileID, runID string) (SandboxReport, error) {
	report := SandboxReport{Writable: []string{}, Denied: []string{}}
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
		report.Denied = sandbox.Denials([]string{text, player}, r.Sandbox, s.home)
		return report, nil
	}
	return report, nil
}
