package installer

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fomod"
)

var _ = Register(fomodInstaller{})

// fomodInstaller resolves a FOMOD's choices into the files it copies, into the package's folder in the mods folder.
type fomodInstaller struct{}

func (fomodInstaller) ID() string { return "fomod" }

func (fomodInstaller) Detect(a Archive, _ Game) bool {
	p, err := fomod.FindConfig(a.Dir)
	return err == nil && p != ""
}

func slash(p string) string { return path.Clean(strings.ReplaceAll(p, `\`, "/")) }

// destination is where op puts src in the package's folder. With no destination the spec installs at the source's own
// path, and an empty one is the package's folder itself; legacy is how Mortar placed both before, at the source's base
// name.
func destination(op fomod.Op, src string, legacy bool) string {
	switch {
	case op.Destination != nil && *op.Destination != "":
		return *op.Destination
	case legacy:
		return path.Base(src)
	case op.Destination == nil:
		return src
	}
	return "."
}

func (fomodInstaller) Layout(a Archive, g Game, choices Choices) (Layout, error) {
	cfg, cfgPath, ok, err := fomod.Open(a.Dir)
	if err != nil || !ok {
		return Layout{}, fmt.Errorf("read the FOMOD config: %w", err)
	}
	// Sources are relative to the folder holding fomod/, which a wrapper folder puts below the archive root.
	wrap, err := filepath.Rel(a.Dir, filepath.Dir(filepath.Dir(cfgPath)))
	if err != nil {
		return Layout{}, err
	}
	wrap = filepath.ToSlash(wrap)
	base := a
	if wrap != "." {
		if base.FS, err = fs.Sub(a.FS, wrap); err != nil {
			return Layout{}, err
		}
	}
	var l Layout
	at := map[string]int{}
	put := func(f File) {
		f.Src = path.Join(wrap, f.Src)
		// Later operations have the higher priority and overwrite earlier ones.
		// Keyed folded: destinations differing only in case are one file on Windows.
		key := strings.ToLower(f.Rel)
		if i, seen := at[key]; seen {
			l.Files[i] = f
			return
		}
		at[key] = len(l.Files)
		l.Files = append(l.Files, f)
	}
	for _, op := range fomod.Resolve(cfg, choices, a.Eval) {
		src := slash(op.Source)
		if !fs.ValidPath(src) {
			return Layout{}, fmt.Errorf("%w: %q", ErrUnsafe, op.Source)
		}
		dest := destination(op, src, a.LegacyFomod)
		// A leading slash means the package's folder, as no slash does.
		dest = slash(cmp.Or(strings.TrimLeft(slash(dest), "/"), "."))
		// A destination is inside the package's own folder: ".." would reach another mod's files.
		if dest == ".." || strings.HasPrefix(dest, "../") {
			return Layout{}, fmt.Errorf("%w: %q", ErrUnsafe, dest)
		}
		src, err := foldedPath(base.FS, src)
		if err != nil {
			return Layout{}, err
		}
		info, err := fs.Stat(base.FS, src)
		if err != nil {
			return Layout{}, err
		}
		if !info.IsDir() && !op.Folder {
			if dest == "." {
				// A file sent to the package's folder itself keeps its own name there.
				dest = path.Base(src)
			}
			put(File{Src: src, Target: TargetMods, Rel: path.Join(a.Key, dest)})
			continue
		}
		inside, err := files(base, src)
		if err != nil {
			return Layout{}, err
		}
		for _, f := range inside {
			put(File{Src: path.Join(src, f), Target: TargetMods, Rel: path.Join(a.Key, dest, f)})
		}
	}
	return l, validate(l, g)
}

// foldedPath resolves p below fsys component by component, taking an exact name when there is one and else the one that
// differs only in case: FOMOD authors wrote for Windows, where "Data" finds "data".
func foldedPath(fsys fs.FS, p string) (string, error) {
	if p == "." {
		return p, nil
	}
	dir := "."
	for _, part := range strings.Split(p, "/") {
		ents, err := fs.ReadDir(fsys, dir)
		if err != nil {
			return "", err
		}
		name := part
		for _, e := range ents {
			if e.Name() == part {
				name = part
				break
			}
			if strings.EqualFold(e.Name(), part) {
				name = e.Name()
			}
		}
		dir = path.Join(dir, name)
	}
	return dir, nil
}
