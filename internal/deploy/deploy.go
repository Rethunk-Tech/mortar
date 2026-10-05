// Package deploy places a loader's install-side files (BepInEx's Doorstop proxy) into a game's folder for a launch and
// removes them again. The profile holds everything else, so nothing else of the game folder is read, moved or taken
// back. Games whose loader redirects the game to the profile's folder (SMAPI's --mods-path) need none of this, so no
// "redirect" deployer is registered.
//
// A launch is Plan, Apply, play, Purge. Apply persists the manifest in the journal folder before it touches the game
// folder, so a crash at any point leaves enough to finish the job: Recover purges from it.
package deploy

import (
	"context"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// View is the profile's side of a deploy.
type View struct {
	// JournalDir holds the manifest and the displaced-file backups.
	JournalDir string `json:"journalDir"`
}

// Op is one file to place.
type Op struct{ Src, Dst string }

// Plan is what Apply will do: copy each Op's Src to Dst, a path in Dir.
type Plan struct {
	Ops  []Op
	Dir  string
	View View
}

// Placed is one placed file as the manifest records it.
type Placed struct {
	Dst  string `json:"dst"`
	Src  string `json:"src"`
	Hash string `json:"hash"`
	// Displaced is where the file that was at Dst before is kept, "" when there was none.
	Displaced string `json:"displaced,omitempty"`
	// Done is set once the file is in place.
	Done bool `json:"done,omitempty"`
	// Undone is set once Purge has taken the file back, so a Purge run again after a failure leaves it alone.
	Undone bool `json:"undone,omitempty"`
}

// Manifest is the persisted record of a deploy.
type Manifest struct {
	Dir  string   `json:"dir"`
	View View     `json:"view"`
	Ops  []Placed `json:"ops"`
	// Created are the folders Apply made, removed again by Purge when empty.
	Created []string `json:"created,omitempty"`
}

// Deployer places a loader's install-side files into the game folder.
type Deployer interface {
	ID() string
	// Plan resolves the loaders' install-side plan files, whose Dst is relative to dir, the install folder.
	Plan(view View, dir string, files []launchplan.PlanFile) (Plan, error)
	// Apply persists the manifest, then places the files.
	Apply(ctx context.Context, p Plan) (Manifest, error)
	// Purge removes the placed files and restores what they displaced, then drops the journal.
	Purge(ctx context.Context, m Manifest) error
	// Recover finishes an unfinished deploy found at journalDir: it does nothing while alive reports the game running,
	// else it purges.
	Recover(ctx context.Context, journalDir string, alive func() bool) error
}

var (
	mu       sync.RWMutex
	registry = map[string]Deployer{}
)

// Register adds a deployer and reports true; a driver calls it in a blank package variable.
func Register(d Deployer) bool {
	mu.Lock()
	defer mu.Unlock()
	registry[d.ID()] = d
	return true
}

// Get returns the registered deployer with this id.
func Get(id string) (Deployer, bool) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := registry[id]
	return d, ok
}
