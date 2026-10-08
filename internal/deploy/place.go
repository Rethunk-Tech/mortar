package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	for _, f := range files {
		if !filepath.IsLocal(f.Dst) {
			return Plan{}, fmt.Errorf("%s leaves the install", f.Dst)
		}
		dst := filepath.Join(dir, f.Dst)
		// Folded so a later file wins over an earlier one that differs only in case, as the file system would.
		byDst[fsx.FoldCase(dst)] = Op{Src: f.Src, Dst: dst}
	}
	plan := Plan{Dir: dir, View: view}
	for _, dst := range slices.Sorted(mapsKeys(byDst)) {
		plan.Ops = append(plan.Ops, byDst[dst])
	}
	return plan, nil
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
	if err := fsx.CheckWritable(p.Dir); err != nil {
		return Manifest{}, err
	}
	m := Manifest{Dir: p.Dir, View: p.View}
	for i, op := range p.Ops {
		hash, err := fsx.SHA256(op.Src)
		if err != nil {
			return Manifest{}, err
		}
		pl := Placed{Dst: op.Dst, Src: op.Src, Hash: hash}
		if _, err := os.Lstat(op.Dst); err == nil {
			pl.Displaced = filepath.Join(p.View.JournalDir, "displaced", fmt.Sprint(i))
		}
		m.Ops = append(m.Ops, pl)
	}
	// The record comes first: from here a crash is recoverable.
	if err := persist(m); err != nil {
		return Manifest{}, err
	}
	for i := range m.Ops {
		if err := ctx.Err(); err != nil {
			return m, err
		}
		if err := put(&m, &m.Ops[i]); err != nil {
			return m, err
		}
		if err := persist(m); err != nil {
			return m, err
		}
	}
	return m, nil
}

func put(m *Manifest, pl *Placed) error {
	if pl.Displaced != "" {
		if err := os.MkdirAll(filepath.Dir(pl.Displaced), 0o700); err != nil {
			return err
		}
		if err := move(pl.Dst, pl.Displaced); err != nil {
			return err
		}
	}
	if err := mkdirTracked(m, filepath.Dir(pl.Dst)); err != nil {
		return err
	}
	if err := datadir.CopyFile(pl.Src, pl.Dst); err != nil {
		return err
	}
	pl.Done = true
	return nil
}

// mkdirTracked creates dir and records the folders it had to make.
func mkdirTracked(m *Manifest, dir string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(d); err == nil || filepath.Dir(d) == d {
			break
		}
		missing = append(missing, d)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	m.Created = append(m.Created, missing...)
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
	for i := len(m.Ops) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		o := m.Ops[i]
		if o.Undone {
			continue
		}
		_, backupErr := os.Lstat(o.Displaced)
		switch {
		case o.Displaced != "" && backupErr == nil:
			// With the player's file set aside, whatever is at Dst is ours, even if the game wrote to it.
			if err := fsx.Remove(o.Dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := move(o.Displaced, o.Dst); err != nil {
				return err
			}
		case o.Displaced == "":
			// Nothing was displaced, so Dst is ours only while it still holds our content.
			if h, err := fsx.SHA256(o.Dst); err == nil && h == o.Hash {
				if err := fsx.Remove(o.Dst); err != nil {
					return err
				}
			}
		}
		// A displaced file with no backup left was never moved or is already back: Dst is the player's own.
		m.Ops[i].Undone = true
		if m.View.JournalDir != "" {
			if err := persist(m); err != nil {
				return err
			}
		}
	}
	// Deepest first: a folder made for one file may hold a sibling's folder made later.
	for _, d := range slices.SortedFunc(slices.Values(m.Created), func(a, b string) int { return len(b) - len(a) }) {
		_ = os.Remove(d)
	}
	return fsx.RemoveAll(m.View.JournalDir)
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
	return p.Purge(ctx, m)
}
