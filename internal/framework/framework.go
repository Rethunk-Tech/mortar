package framework

import (
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Input is the profile a framework analyzes: All its mods, and the Enabled ones among them.
type Input struct{ Enabled, All []Mod }

// Findings is what a framework's analyzer adds to a profile's problems.
type Findings struct {
	AssetConflicts []AssetConflict
	Settings       []SettingHint
	Cleanup        []Cleanup
	Redundant      []Redundant
}

// Framework is a framework mod's driver, keyed by the framework's mod id.
type Framework interface {
	ID() mod.ID
	// Matches reports whether m is the framework or content written for it, which turns the framework's analyzers on.
	Matches(m Mod) bool
	Analyze(in Input) Findings
}

var (
	mu       sync.RWMutex
	registry = map[string]Framework{}
)

// Register adds a framework and reports true; a driver package calls it in a blank package variable
// (var _ = framework.Register(...)), so importing the package registers it.
func Register(f Framework) bool {
	mu.Lock()
	defer mu.Unlock()
	registry[f.ID().Fold()] = f
	return true
}

// Present lists the registered frameworks that an enabled mod of the profile matches, ordered by id.
func Present(enabled []Mod) []Framework {
	mu.RLock()
	defer mu.RUnlock()
	var out []Framework
	for _, f := range registry {
		if slices.ContainsFunc(enabled, f.Matches) {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b Framework) int { return strings.Compare(a.ID().Fold(), b.ID().Fold()) })
	return out
}

// Forgetter is a framework that keeps state between analyses.
type Forgetter interface{ Forget() }

// Forget drops what every registered framework keeps between analyses.
func Forget() {
	mu.RLock()
	defer mu.RUnlock()
	for _, f := range registry {
		if g, ok := f.(Forgetter); ok {
			g.Forget()
		}
	}
}

// AssetIndexer is a framework whose packs edit game assets by name, so a profile can map which mods change each asset.
type AssetIndexer interface{ IndexesAssets() }

// IndexesAssets reports whether a registered framework for one of formats maps its packs' assets.
func IndexesAssets(formats []string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, f := range registry {
		if _, ok := f.(AssetIndexer); ok && slices.Contains(formats, f.ID().Format()) {
			return true
		}
	}
	return false
}

// HasFrameworks reports whether a registered framework is written for mods of one of formats, so a loader's mods can
// be grouped by the framework they are content for.
func HasFrameworks(formats []string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, f := range registry {
		if slices.Contains(formats, f.ID().Format()) {
			return true
		}
	}
	return false
}
