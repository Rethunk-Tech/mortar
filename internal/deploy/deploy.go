// Package deploy places a profile's files into a game's folders for a launch, takes back what the game wrote, and
// removes the placed files again. Games whose loader redirects the game to the profile's folder (SMAPI's --mods-path)
// need none of this, so no "redirect" deployer is registered.
//
// A launch is Plan, Apply, play, Harvest, Purge. Apply persists the manifest in the journal folder before it touches
// the game folder, so a crash at any point leaves enough to finish the job: Recover harvests and purges from it.
package deploy

import (
	"context"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// Target is a content target in the install. Writable targets receive copies, never links, because the game writes
// to them; what the game writes there returns to Home on harvest.
type Target struct {
	ID       string `json:"id"`
	Root     string `json:"root"`
	Writable bool   `json:"writable,omitempty"`
	// Home is the profile-side folder a writable target's files return to.
	Home string `json:"home,omitempty"`
}

// View is the profile's side of a deploy.
type View struct {
	// JournalDir holds the manifest and the displaced-file backups.
	JournalDir string `json:"journalDir"`
	// Overwrite receives what the game wrote outside a writable target.
	Overwrite string `json:"overwrite,omitempty"`
}

// InstallView is the game's side of a deploy: the install folder (where launch plan files go) and its targets.
type InstallView struct {
	Dir     string
	Targets []Target
}

// Package is one package's layout and the folder its Src paths are relative to.
type Package struct {
	ID     string
	Root   string
	Layout installer.Layout
}

// Op is one file to place.
type Op struct {
	Package  string
	Src, Dst string
	Target   string
	Writable bool
}

// Conflict is a destination several packages provide; the last in package order wins.
type Conflict struct {
	Dst             string
	Winners, Losers []string
}

// Plan is what Apply will do.
type Plan struct {
	Ops       []Op
	Conflicts []Conflict
	Dir       string
	Targets   []Target
	View      View
}

// Placed is one placed file as the manifest records it.
type Placed struct {
	Dst      string `json:"dst"`
	Src      string `json:"src"`
	Hash     string `json:"hash"`
	Writable bool   `json:"writable,omitempty"`
	// Displaced is where the file that was at Dst before is kept, "" when there was none.
	Displaced string `json:"displaced,omitempty"`
	// Done is set once the file is in place.
	Done bool `json:"done,omitempty"`
}

// Manifest is the persisted record of a deploy.
type Manifest struct {
	Dir     string   `json:"dir"`
	Targets []Target `json:"targets"`
	View    View     `json:"view"`
	Ops     []Placed `json:"ops"`
	// Created are the folders Apply made, removed again by Purge when empty.
	Created []string `json:"created,omitempty"`
	// Existing are the files in the target roots before Apply, which harvest must leave alone.
	Existing []string `json:"existing,omitempty"`
}

// Change is something harvest took back.
type Change struct {
	Path string
	// Dest is where it went.
	Dest string
	// Kind is new or changed.
	Kind string
}

// Deployer places files into an install.
type Deployer interface {
	ID() string
	// Plan orders the packages (lowest priority first) and resolves conflicts; files are the loaders' install-side
	// plan files, which sit under every package.
	Plan(view View, inst InstallView, pkgs []Package, files []launchplan.PlanFile) (Plan, error)
	// Apply persists the manifest, then places the files.
	Apply(ctx context.Context, p Plan) (Manifest, error)
	// Harvest takes back what the game wrote once it has exited.
	Harvest(ctx context.Context, m Manifest) ([]Change, error)
	// Purge removes the placed files and restores what they displaced, then drops the journal.
	Purge(ctx context.Context, m Manifest) error
	// Recover finishes an unfinished deploy found at journalDir: it does nothing while alive reports the game running,
	// else it harvests and purges.
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
