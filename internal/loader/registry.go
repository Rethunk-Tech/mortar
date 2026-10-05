package loader

import (
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

var (
	mu       sync.RWMutex
	registry = map[string]Loader{}
)

// Register adds a loader and reports true; a driver package calls it in a blank package variable
// (var _ = loader.Register(...)), so importing the package registers it.
func Register(l Loader) bool {
	mu.Lock()
	defer mu.Unlock()
	registry[l.ID()] = l
	return true
}

// Get returns the registered loader with this id.
func Get(id string) (Loader, bool) {
	mu.RLock()
	defer mu.RUnlock()
	l, ok := registry[strings.ToLower(strings.TrimSpace(id))]
	return l, ok
}

// All lists every registered loader ordered by id.
func All() []Loader {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Loader, 0, len(registry))
	for _, l := range registry {
		out = append(out, l)
	}
	slices.SortFunc(out, func(a, b Loader) int { return strings.Compare(a.ID(), b.ID()) })
	return out
}

// For lists the game's loaders in catalog order; a catalog entry without a registered driver is left out.
func For(g components.GameInfo) []Loader {
	var out []Loader
	for _, gl := range g.Loaders {
		if l, ok := Get(gl.ID); ok {
			out = append(out, l)
		}
	}
	return out
}
