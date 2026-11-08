// Package contentpatcher is the Content Patcher framework: it reads the content packs written for it to find asset
// conflicts, shadowed packs, config settings that fight the profile's mods, and unused tilesheet packs.
package contentpatcher

import (
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// ID is Content Patcher's mod id.
var ID = mod.SMAPI("Pathoschild.ContentPatcher")

var _ = framework.Register(Driver{})

// Driver is the Content Patcher framework.
type Driver struct{}

func (Driver) ID() mod.ID { return ID }

// IndexesAssets marks Content Patcher as the framework behind the asset map.
func (Driver) IndexesAssets() {}

// Matches is Content Patcher itself and every pack written for it.
func (Driver) Matches(m framework.Mod) bool {
	return mod.Equal(m.ModID(), ID) || cp.IsContentPatcherPack(cpMod(m))
}

// Analyze reads every enabled pack once; the tilesheet cleanup also covers disabled packs.
func (Driver) Analyze(in framework.Input) framework.Findings {
	cp.SetCacheRoot(loadCacheRoot())
	req := cpRequest(in)
	req.Extend = func(run *cp.PartsRun, all []cp.Mod) cp.Extension {
		return cp.Extension{Hints: compatibilitySettings(cpMods(in.Enabled), run, in.SkipImageOverlap)}
	}
	found := cp.Analyze(req)
	out := toFrameworkFindings(found)
	out.Cleanup = append(tilesheetCleanups(found.Tilesheets, found.TilesheetsUnreadable), recolourAddons(cpMods(in.All))...)
	return out
}

// Forget drops what the driver keeps between checks.
func (Driver) Forget() { cp.Forget() }

func loadCacheRoot() string {
	dir, err := datadir.Dir()
	if err != nil {
		return ""
	}
	return dir
}
