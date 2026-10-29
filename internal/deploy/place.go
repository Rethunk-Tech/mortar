package deploy

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

const journalFile = "deploy.json"

// Place copies a loader's install-side files into the game folder. The game and its anti-cheat may write to the placed
// files, so they are copies, never links into the store.
type Place struct{}

// Plan resolves the loaders' install-side plan files, whose Dst is relative to dir, the install folder.
func (Place) Plan(view View, dir string, files []launchplan.PlanFile) (Plan, error) {
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
	return datadir.WriteJSONIn(m.View.JournalDir, journalFile, m)
}

// ErrUnrecovered is Apply's refusal to deploy over the journal of a deploy that was never taken back: a second
// manifest would overwrite the first's record of the player's own files, and they would be lost.
var ErrUnrecovered = errors.New("an earlier deploy has not been taken back")

// Apply persists the manifest, then places the files.
func (Place) Apply(ctx context.Context, p Plan) (Manifest, error) {
	if _, err := os.Lstat(journalPath(p.View.JournalDir)); err == nil {
		return Manifest{}, fmt.Errorf("%w: recover %s first", ErrUnrecovered, p.View.JournalDir)
	}
	// A plan that places only into a folder outside the install (a mods folder in Documents) needs no write access to it.
	if slices.ContainsFunc(p.Ops, func(o Op) bool { return datadir.UnderRoot(p.Dir, o.Dst) }) {
		if err := fsx.CheckWritable(p.Dir); err != nil {
			return Manifest{}, err
		}
	}
	m := Manifest{Dir: p.Dir, View: p.View, Owned: p.Owned, Existing: listOwned(p.Owned)}
	for i, op := range p.Ops {
		hash, err := fsx.HashFile(op.Src, sha256.New())
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
	return fsx.Remove(src)
}

// Purge removes the placed files and restores what they displaced, then drops the journal.
func (Place) Purge(ctx context.Context, m Manifest) error {
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
		_ = fsx.Remove(d)
		at("folder:" + d)
	}
	err := log.close()
	at("journal")
	return errors.Join(err, saveNotes(m.View.JournalDir, m.notes), fsx.RemoveAll(m.View.JournalDir))
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
			if err := writeBackOrRescue(m, o, i, log); err != nil {
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
			if h, err := fsx.HashFile(o.Dst, sha256.New()); err == nil {
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
					} else {
						m.notes = append(m.notes, Note{Path: o.Dst, Kind: NoteRescued})
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

// writeBackOrRescue copies a changed Dst of a profile-copy file over its Src before Dst is removed and the displaced file
// returns. When the profile's folder is gone it keeps the bytes beside Dst under a rescue name, on the log first, so a
// changed file is never removed without a copy.
func writeBackOrRescue(m *Manifest, o Placed, i int, l *opLog) error {
	// A link a mod left in place of the file is not followed into the profile; removing it loses no bytes.
	if st, err := os.Lstat(o.Dst); !o.WriteBack || (err == nil && !st.Mode().IsRegular()) {
		return nil
	}
	h, err := fsx.HashFile(o.Dst, sha256.New())
	if errors.Is(err, fs.ErrNotExist) || (err == nil && h == o.Hash) {
		return nil
	}
	if err != nil {
		return err
	}
	// Dst holding the backup's bytes is the player's file half moved (a copy across file systems cut short), not ours.
	if b, err := fsx.HashFile(o.Displaced, sha256.New()); err == nil && b == h {
		return nil
	}
	wrote, err := writeBackFile(o, i, l)
	if err != nil || wrote {
		return err
	}
	rescue := RescueName(o.Dst)
	if err := l.add(step{R: rescue}, true); err != nil {
		return err
	}
	at(fmt.Sprintf("rescue-logged:%d", i))
	log.Printf("deploy: %s could not go back into its profile; its bytes are kept as %s", o.Dst, rescue)
	if err := fsx.Rename(o.Dst, rescue); err != nil {
		return err
	}
	at(fmt.Sprintf("rescued:%d", i))
	m.notes = append(m.notes, Note{Path: rescue, Kind: NoteRescued})
	return nil
}

// RescueName is where the bytes of a changed file are kept when its profile is gone.
func RescueName(dst string) string { return dst + ".mortar-rescued" }

// writeBackFile copies Dst over Src through a temp file and a rename, so the bytes are in both places until Dst is
// removed. It reports false, leaving Dst alone, when the profile's folder is gone.
func writeBackFile(o Placed, i int, log *opLog) (bool, error) {
	if _, err := os.Stat(filepath.Dir(o.Src)); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := datadir.CopyFile(o.Dst, o.Src); err != nil {
		return false, err
	}
	at(fmt.Sprintf("written-back:%d", i))
	return true, log.add(step{W: &i}, false)
}

// Recover finishes an unfinished deploy found at journalDir: it does nothing while alive reports the game running,
// else it purges.
func (p Place) Recover(ctx context.Context, journalDir string, alive func() bool) error {
	var m Manifest
	found, err := datadir.ReadJournal(journalPath(journalDir), alive, &m)
	if err != nil {
		return fmt.Errorf("read the deploy journal: %w", err)
	}
	if !found {
		return nil
	}
	// The journal may have moved with the data folder since it was written.
	if old := m.View.JournalDir; old != journalDir {
		for i := range m.Ops {
			if rel, err := filepath.Rel(old, m.Ops[i].Displaced); old != "" && m.Ops[i].Displaced != "" && err == nil && filepath.IsLocal(rel) {
				m.Ops[i].Displaced = filepath.Join(journalDir, rel)
			}
		}
		m.View.JournalDir = journalDir
	}
	if err := replay(&m); err != nil {
		return fmt.Errorf("read the deploy log: %w", err)
	}
	return p.Purge(ctx, m)
}
