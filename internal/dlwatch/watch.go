package dlwatch

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	pollEvery = 3 * time.Second
	stableFor = 2 * time.Second
)

type seenFile struct {
	size    int64
	stable  time.Time
	ready   bool
	ignored bool // existed when watching started
}

// Folder polls dir for new finished archives. Stable is overridable in tests.
type Folder struct {
	Dir    string
	Stable time.Duration
	Now    func() time.Time

	known   map[string]*seenFile
	started bool
}

func (f *Folder) stable() time.Duration {
	if f.Stable > 0 {
		return f.Stable
	}
	return stableFor
}

func (f *Folder) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func siblingPartial(dir, name string) bool {
	for _, suf := range []string{".crdownload", ".part"} {
		if _, err := os.Stat(filepath.Join(dir, name+suf)); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+suf)); err == nil {
			return true
		}
	}
	return false
}

// Snapshot records names already in dir so the next Poll ignores them.
func (f *Folder) Snapshot() error {
	if f.known == nil {
		f.known = map[string]*seenFile{}
	}
	ents, err := os.ReadDir(f.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !archiveExt(e.Name()) {
			continue
		}
		f.known[e.Name()] = &seenFile{ignored: true, ready: true}
	}
	return nil
}

// Poll returns archives that appeared after Snapshot, are not partial, and
// whose size has been unchanged for Stable.
func (f *Folder) Poll() []string {
	if f.known == nil {
		f.known = map[string]*seenFile{}
	}
	ents, err := os.ReadDir(f.Dir)
	if err != nil {
		return nil
	}
	now := f.now()
	present := map[string]struct{}{}
	var out []string
	for _, e := range ents {
		if e.IsDir() || !archiveExt(e.Name()) {
			continue
		}
		present[e.Name()] = struct{}{}
		info, err := e.Info()
		if err != nil {
			continue
		}
		prev, ok := f.known[e.Name()]
		if !ok {
			f.known[e.Name()] = &seenFile{size: info.Size(), stable: now}
			continue
		}
		if prev.ignored || prev.ready {
			continue
		}
		if siblingPartial(f.Dir, e.Name()) {
			prev.size = info.Size()
			prev.stable = now
			continue
		}
		if info.Size() != prev.size {
			prev.size = info.Size()
			prev.stable = now
			continue
		}
		if now.Sub(prev.stable) < f.stable() {
			continue
		}
		prev.ready = true
		out = append(out, filepath.Join(f.Dir, e.Name()))
	}
	for name := range f.known {
		if _, ok := present[name]; !ok && !f.known[name].ignored {
			delete(f.known, name)
		}
	}
	return out
}
