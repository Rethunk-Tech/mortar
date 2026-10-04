// Package folderwatch tells the window when a folder the library reads changes, so its lists refetch.
package folderwatch

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event names, one per kind of folder.
const (
	ModsFolderEvent  = "library:mods-folder"
	ExtraFolderEvent = "library:extra-folder"
	DownloadsEvent   = "library:downloads"
)

// Target is one folder to watch; Event is emitted with Game after a burst of changes in it.
type Target struct {
	Event string
	Game  string
	Dir   string
}

// Deps wires the watcher to the current targets and the event bus.
type Deps struct {
	Targets func() []Target
	Emit    func(name string, data any)
	// Quiet is how long a folder stays unchanged before its event fires.
	Quiet time.Duration
	// Retarget is how often Targets is re-read, which also retries folders that did not exist.
	Retarget time.Duration
}

type watch struct {
	t     Target
	timer *time.Timer
}

// Run watches until ctx ends. fsnotify watches each folder's top level only; a folder that is missing is retried at
// every retarget tick and fires once when it appears.
func Run(ctx context.Context, d Deps) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	quiet, every := d.Quiet, d.Retarget
	if quiet <= 0 {
		quiet = 500 * time.Millisecond
	}
	if every <= 0 {
		every = 2 * time.Second
	}
	var mu sync.Mutex
	active := map[string]*watch{} // by Dir
	missing := map[string]bool{}  // folders seen absent, so their appearing fires once
	fire := func(t Target) { d.Emit(t.Event, t.Game) }
	retarget := func() {
		mu.Lock()
		defer mu.Unlock()
		want := map[string]Target{}
		for _, t := range d.Targets() {
			if t.Dir != "" {
				want[t.Dir] = t
			}
		}
		for dir, a := range active {
			if t, ok := want[dir]; !ok || t != a.t {
				a.timer.Stop()
				_ = w.Remove(dir)
				delete(active, dir)
			}
		}
		for dir, t := range want {
			if active[dir] != nil {
				continue
			}
			if st, err := os.Stat(dir); err != nil || !st.IsDir() {
				missing[dir] = true
				continue
			}
			if err := w.Add(dir); err != nil {
				log.Printf("folderwatch: %s: %v", dir, err)
				continue
			}
			a := &watch{t: t, timer: time.AfterFunc(quiet, func() { fire(t) })}
			if !missing[dir] {
				a.timer.Stop()
			}
			delete(missing, dir)
			active[dir] = a
		}
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	retarget()
	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, a := range active {
				a.timer.Stop()
			}
			mu.Unlock()
			return nil
		case <-tick.C:
			retarget()
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			mu.Lock()
			if a := active[ev.Name]; a != nil && ev.Has(fsnotify.Remove|fsnotify.Rename) {
				// The watched folder itself went away: drop it so a later tick re-adds it when it returns.
				a.timer.Reset(quiet)
				_ = w.Remove(ev.Name)
				delete(active, ev.Name)
				missing[ev.Name] = true
			} else if a := active[filepath.Dir(ev.Name)]; a != nil {
				a.timer.Reset(quiet)
			}
			mu.Unlock()
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("folderwatch: %v", err)
		}
	}
}
