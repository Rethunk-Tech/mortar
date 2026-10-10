package deploy

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// ID of the deployer that copies files into the install itself.
const copyID = "copy-into-install"

var _ = Register(place{})

const journalFile = "deploy.json"

// place copies files into the install. The game and its anti-cheat may write to the placed files, so they are copies,
// never links into the store.
type place struct{}

func (place) ID() string { return copyID }

func (place) Plan(view View, dir string, files []launchplan.PlanFile) (Plan, error) {
	byDst := map[string]Op{}
	var owned []Owned
	for _, f := range files {
		base := dir
		if f.Root != "" {
			if base = view.Roots[f.Root]; base == "" || !filepath.IsAbs(base) {
				return Plan{}, fmt.Errorf("no %s folder to place %s in", f.Root, f.Dst)
			}
		}
		if !filepath.IsLocal(f.Dst) {
			return Plan{}, fmt.Errorf("%s leaves the %s folder", f.Dst, cmp.Or(f.Root, "install"))
		}
		dst := filepath.Join(base, f.Dst)
		// Folded so a later file wins over an earlier one that differs only in case, as the file system would.
		byDst[fsx.FoldCase(dst)] = Op{Src: f.Src, Dst: dst, WriteBack: f.Root != ""}
		if o, ok := ownedOf(base, f); ok && !slices.Contains(owned, o) {
			owned = append(owned, o)
		}
	}
	plan := Plan{Dir: dir, View: view, Owned: owned}
	for _, dst := range slices.Sorted(mapsKeys(byDst)) {
		plan.Ops = append(plan.Ops, byDst[dst])
	}
	return plan, nil
}

// ownedOf is the top folder of a path role's file, which its entry owns, paired with the profile's folder of that name
// (the plan file's Src ends with its Dst). A file at the role's root has none.
func ownedOf(base string, f launchplan.PlanFile) (Owned, bool) {
	top, _, nested := strings.Cut(filepath.ToSlash(f.Dst), "/")
	if f.Root == "" || !nested || !strings.HasSuffix(f.Src, f.Dst) {
		return Owned{}, false
	}
	profileRoot := strings.TrimSuffix(f.Src, f.Dst)
	return Owned{Dst: filepath.Join(base, top), Src: filepath.Join(profileRoot, top)}, true
}

func mapsKeys[V any](m map[string]V) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func journalPath(dir string) string { return filepath.Join(dir, journalFile) }

// HasJournal reports whether dir holds the record of a deploy not yet undone.
func HasJournal(dir string) bool {
	_, err := os.Stat(journalPath(dir))
	return err == nil
}

func persist(m Manifest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.View.JournalDir, 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(journalPath(m.View.JournalDir), b, 0o600)
}

// ErrUnrecovered is Apply's refusal to deploy over the journal of a deploy that was never taken back: a second
// manifest would overwrite the first's record of the player's own files, and they would be lost.
var ErrUnrecovered = errors.New("an earlier deploy has not been taken back")

func (place) Apply(ctx context.Context, p Plan) (Manifest, error) {
	if _, err := os.Lstat(journalPath(p.View.JournalDir)); err == nil {
		return Manifest{}, fmt.Errorf("%w: recover %s first", ErrUnrecovered, p.View.JournalDir)
	}
	// A plan that places only into a folder outside the install (a mods folder in Documents) needs no write access to it.
	if slices.ContainsFunc(p.Ops, func(o Op) bool { return inside(p.Dir, o.Dst) }) {
		if err := fsx.CheckWritable(p.Dir); err != nil {
			return Manifest{}, err
		}
	}
	m := Manifest{Dir: p.Dir, View: p.View, Owned: p.Owned, Started: time.Now().UnixNano()}
	for i, op := range p.Ops {
		hash, err := fsx.SHA256(op.Src)
		if err != nil {
			return Manifest{}, err
		}
		pl := Placed{Dst: op.Dst, Src: op.Src, Hash: hash, WriteBack: op.WriteBack}
		if _, err := os.Lstat(op.Dst); err == nil {
			pl.Displaced = filepath.Join(p.View.JournalDir, "displaced", fmt.Sprint(i))
		}
		m.Ops = append(m.Ops, pl)
	}
	// The record comes first: from here a crash is recoverable.
	_ = fsx.Remove(logPath(p.View.JournalDir))
	if err := persist(m); err != nil {
		return Manifest{}, err
	}
	at("record")
	log, err := openLog(p.View.JournalDir)
	if err != nil {
		return Manifest{}, err
	}
	for i := range m.Ops {
		if err := ctx.Err(); err != nil {
			return m, errors.Join(err, log.close())
		}
		if err := put(&m, i, log); err != nil {
			return m, errors.Join(err, log.close())
		}
	}
	return m, errors.Join(log.close(), persist(m))
}

func inside(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && filepath.IsLocal(rel)
}

func put(m *Manifest, i int, log *opLog) error {
	pl := &m.Ops[i]
	n := fmt.Sprint(i)
	if pl.Displaced != "" {
		if err := os.MkdirAll(filepath.Dir(pl.Displaced), 0o700); err != nil {
			return err
		}
		if err := move(pl.Dst, pl.Displaced); err != nil {
			return err
		}
		at("displaced:" + n)
	}
	if err := mkdirTracked(m, filepath.Dir(pl.Dst), log, n); err != nil {
		return err
	}
	at("before-copy:" + n)
	if err := datadir.CopyFile(pl.Src, pl.Dst); err != nil {
		return err
	}
	at("copied:" + n)
	pl.Done = true
	if err := log.add(step{P: &i}, false); err != nil {
		return err
	}
	at("logged:" + n)
	return nil
}

// mkdirTracked creates dir and records the folders it had to make. Each is on the log, flushed, before it exists, so a
// crash cannot strand a folder Mortar made.
func mkdirTracked(m *Manifest, dir string, log *opLog, n string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(d); err == nil || filepath.Dir(d) == d {
			break
		}
		missing = append(missing, d)
	}
	if len(missing) == 0 {
		return nil
	}
	for _, d := range missing {
		if err := log.add(step{M: d}, true); err != nil {
			return err
		}
	}
	at("mkdir-logged:" + n)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	m.Created = append(m.Created, missing...)
	at("mkdir:" + n)
	return nil
}

// move renames, falling back to copy and remove across file systems.
func move(src, dst string) error {
	if err := fsx.Rename(src, dst); err == nil {
		return nil
	}
	if err := datadir.CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func (place) Purge(ctx context.Context, m Manifest) error {
	var log *opLog
	if m.View.JournalDir != "" {
		var err error
		if log, err = openLog(m.View.JournalDir); err != nil {
			return err
		}
	}
	if err := undoOps(ctx, &m, log); err != nil {
		return errors.Join(err, log.close())
	}
	at("ops-undone")
	if err := adopt(&m, log); err != nil {
		return errors.Join(err, log.close())
	}
	at("adopted")
	// Deepest first: a folder made for one file may hold a sibling's folder made later.
	for _, d := range slices.SortedFunc(slices.Values(m.Created), func(a, b string) int { return len(b) - len(a) }) {
		_ = os.Remove(d)
		at("folder:" + d)
	}
	err := log.close()
	at("journal")
	return errors.Join(err, fsx.RemoveAll(m.View.JournalDir))
}

func undoOps(ctx context.Context, m *Manifest, log *opLog) error {
	for i := len(m.Ops) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		o := m.Ops[i]
		if o.Undone {
			continue
		}
		// A crashed copy leaves its temp file; the name is Mortar's, so it goes whatever else is known of the operation.
		if err := fsx.Remove(tmpName(o.Dst)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		_, backupErr := os.Lstat(o.Displaced)
		switch {
		case o.Displaced != "" && backupErr == nil:
			// With the player's file set aside, whatever is at Dst is ours, even if the game wrote to it.
			if err := writeBack(o, i, log); err != nil {
				return err
			}
			if err := fsx.Remove(o.Dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := move(o.Displaced, o.Dst); err != nil {
				return err
			}
		case o.Displaced == "":
			// Nothing was displaced, so Dst is ours only while it still holds our content. A file changed since holds
			// the player's or a mod's own data (a script mod's settings): one placed from the profile's copy goes back
			// into that copy, and any other stays.
			if h, err := fsx.SHA256(o.Dst); err == nil {
				switch {
				case h == o.Hash:
					if err := fsx.Remove(o.Dst); err != nil {
						return err
					}
				case o.WriteBack:
					wrote, err := writeBackFile(o, i, log)
					if err != nil {
						return err
					}
					if wrote {
						if err := fsx.Remove(o.Dst); err != nil {
							return err
						}
					}
				}
			}
		}
		// A displaced file with no backup left was never moved or is already back: Dst is the player's own.
		at(fmt.Sprintf("undone-file:%d", i))
		m.Ops[i].Undone = true
		if err := log.add(step{U: &i}, false); err != nil {
			return err
		}
		at(fmt.Sprintf("undone-logged:%d", i))
	}
	return nil
}

// writeBack copies a changed Dst of a profile-copy file over its Src before Dst is removed or the displaced file returns.
func writeBack(o Placed, i int, log *opLog) error {
	if !o.WriteBack {
		return nil
	}
	if h, err := fsx.SHA256(o.Dst); err != nil || h == o.Hash {
		return nil
	}
	_, err := writeBackFile(o, i, log)
	return err
}

// writeBackFile copies Dst over Src through a temp file and a rename, so the bytes are in both places until Dst is
// removed. It reports false, leaving Dst alone, when the profile's folder is gone.
func writeBackFile(o Placed, i int, log *opLog) (bool, error) {
	if _, err := os.Stat(filepath.Dir(o.Src)); err != nil {
		return false, nil
	}
	if err := datadir.CopyFile(o.Dst, o.Src); err != nil {
		return false, err
	}
	at(fmt.Sprintf("written-back:%d", i))
	return true, log.add(step{W: &i}, false)
}

func (p place) Recover(ctx context.Context, journalDir string, alive func() bool) error {
	b, err := fsx.ReadFile(journalPath(journalDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if alive != nil && alive() {
		return nil
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("read the deploy journal: %w", err)
	}
	if strings.TrimSpace(m.View.JournalDir) == "" {
		m.View.JournalDir = journalDir
	}
	if err := replay(&m); err != nil {
		return fmt.Errorf("read the deploy log: %w", err)
	}
	return p.Purge(ctx, m)
}
