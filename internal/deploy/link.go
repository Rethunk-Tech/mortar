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

// ID of the deployer that puts files into the install itself.
const linkID = "link-into-install"

var _ = Register(link{})

const journalFile = "deploy.json"

// link hardlinks files into the install, copying where a link is impossible (another file system) or unwanted
// (writable targets).
type link struct{}

func (link) ID() string { return linkID }

func (link) Plan(view View, inst InstallView, pkgs []Package, files []launchplan.PlanFile) (Plan, error) {
	type source struct {
		op  Op
		who []string
	}
	byDst := map[string]*source{}
	add := func(op Op) {
		if s, ok := byDst[op.Dst]; ok {
			s.op = op
			s.who = append(s.who, op.Package)
			return
		}
		byDst[op.Dst] = &source{op: op, who: []string{op.Package}}
	}
	for _, f := range files {
		add(Op{Package: "loader", Src: f.Src, Dst: filepath.Join(inst.Dir, f.Dst)})
	}
	for _, p := range pkgs {
		for _, f := range p.Layout.Files {
			i := slices.IndexFunc(inst.Targets, func(t Target) bool { return t.ID == f.Target })
			if i < 0 {
				return Plan{}, fmt.Errorf("%s: no target %q in the install", p.ID, f.Target)
			}
			t := inst.Targets[i]
			add(Op{Package: p.ID, Src: filepath.Join(p.Root, filepath.FromSlash(f.Src)), Dst: filepath.Join(t.Root, filepath.FromSlash(f.Rel)), Target: t.ID, Writable: t.Writable})
		}
	}
	plan := Plan{Dir: inst.Dir, Targets: inst.Targets, View: view}
	for _, dst := range slices.Sorted(mapsKeys(byDst)) {
		s := byDst[dst]
		plan.Ops = append(plan.Ops, s.op)
		if len(s.who) > 1 {
			plan.Conflicts = append(plan.Conflicts, Conflict{Dst: dst, Winners: s.who[len(s.who)-1:], Losers: s.who[:len(s.who)-1]})
		}
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

func persist(m Manifest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.View.JournalDir, 0o700); err != nil {
		return err
	}
	tmp := journalPath(m.View.JournalDir) + ".tmp"
	if err := fsx.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return fsx.Rename(tmp, journalPath(m.View.JournalDir))
}

func (link) Apply(ctx context.Context, p Plan) (Manifest, error) {
	m := Manifest{Dir: p.Dir, Targets: p.Targets, View: p.View}
	for _, t := range p.Targets {
		_ = filepath.WalkDir(t.Root, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				m.Existing = append(m.Existing, path)
			}
			return nil
		})
	}
	for i, op := range p.Ops {
		hash, err := fsx.SHA256(op.Src)
		if err != nil {
			return Manifest{}, err
		}
		pl := Placed{Dst: op.Dst, Src: op.Src, Hash: hash, Writable: op.Writable}
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
		if err := place(&m, &m.Ops[i]); err != nil {
			return m, err
		}
	}
	return m, persist(m)
}

func place(m *Manifest, pl *Placed) error {
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
	if !pl.Writable && os.Link(pl.Src, pl.Dst) == nil {
		pl.Done = true
		return nil
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
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := datadir.CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func (link) Harvest(ctx context.Context, m Manifest) ([]Change, error) {
	known := map[string]Placed{}
	for _, o := range m.Ops {
		known[o.Dst] = o
	}
	existing := map[string]bool{}
	for _, p := range m.Existing {
		existing[p] = true
	}
	var changes []Change
	for _, t := range m.Targets {
		err := filepath.WalkDir(t.Root, func(path string, d fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			rel, _ := filepath.Rel(t.Root, path)
			if o, ok := known[path]; ok {
				h, err := fsx.SHA256(path)
				if errors.Is(err, fs.ErrNotExist) || (err == nil && h == o.Hash) {
					return nil
				}
				if err != nil {
					return err
				}
				c, err := take(m, t, rel, path, false)
				if c != nil {
					c.Kind = "changed"
					changes = append(changes, *c)
				}
				return err
			}
			if existing[path] {
				return nil
			}
			c, err := take(m, t, rel, path, true)
			if c != nil {
				c.Kind = "new"
				changes = append(changes, *c)
			}
			return err
		})
		if err != nil {
			return changes, err
		}
	}
	return changes, nil
}

// take sends a file the game wrote to its home: a writable target's own folder, else the overwrite folder. A new
// file is moved, so purge leaves nothing behind; a changed placed file is copied, since purge removes it anyway.
func take(m Manifest, t Target, rel, path string, relocate bool) (*Change, error) {
	var dest string
	switch {
	case t.Writable && t.Home != "":
		dest = filepath.Join(t.Home, rel)
	case m.View.Overwrite != "":
		dest = filepath.Join(m.View.Overwrite, t.ID, rel)
	default:
		return nil, fmt.Errorf("%s was written by the game and has no folder to go to", path)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return nil, err
	}
	if relocate {
		_ = os.Remove(dest)
		if err := move(path, dest); err != nil {
			return nil, err
		}
	} else {
		_ = os.Remove(dest)
		if err := datadir.CopyFile(path, dest); err != nil {
			return nil, err
		}
	}
	return &Change{Path: path, Dest: dest}, nil
}

func (link) Purge(ctx context.Context, m Manifest) error {
	for _, o := range slices.Backward(m.Ops) {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, backupErr := os.Lstat(o.Displaced)
		hasBackup := o.Displaced != "" && backupErr == nil
		switch {
		case hasBackup:
			if err := os.Remove(o.Dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := move(o.Displaced, o.Dst); err != nil {
				return err
			}
		default:
			if h, err := fsx.SHA256(o.Dst); err == nil && (h == o.Hash || o.Done) {
				if err := os.Remove(o.Dst); err != nil {
					return err
				}
			}
		}
	}
	for _, d := range slices.Backward(m.Created) {
		_ = os.Remove(d)
	}
	return os.RemoveAll(m.View.JournalDir)
}

func (l link) Recover(ctx context.Context, journalDir string, alive func() bool) error {
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
	if _, err := l.Harvest(ctx, m); err != nil {
		return err
	}
	return l.Purge(ctx, m)
}
