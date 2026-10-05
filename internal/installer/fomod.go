package installer

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
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

func (fomodInstaller) Layout(a Archive, g Game, choices Choices) (Layout, error) {
	cfg, _, ok, err := fomod.Open(a.Dir)
	if err != nil || !ok {
		return Layout{}, fmt.Errorf("read the FOMOD config: %w", err)
	}
	var l Layout
	at := map[string]int{}
	put := func(f File) {
		// Later operations have the higher priority and overwrite earlier ones.
		if i, seen := at[f.Rel]; seen {
			l.Files[i] = f
			return
		}
		at[f.Rel] = len(l.Files)
		l.Files = append(l.Files, f)
	}
	for _, op := range fomod.Resolve(cfg, choices, a.Eval) {
		src := slash(op.Source)
		if !fs.ValidPath(src) {
			return Layout{}, fmt.Errorf("%w: %q", ErrUnsafe, op.Source)
		}
		dest := op.Destination
		if dest == "" {
			dest = path.Base(src)
		}
		// A leading slash means the package's folder, as no slash does.
		dest = slash(cmp.Or(strings.TrimLeft(slash(dest), "/"), "."))
		// A destination is inside the package's own folder: ".." would reach another mod's files.
		if dest == ".." || strings.HasPrefix(dest, "../") {
			return Layout{}, fmt.Errorf("%w: %q", ErrUnsafe, op.Destination)
		}
		info, err := fs.Stat(a.FS, src)
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
		inside, err := files(a, src)
		if err != nil {
			return Layout{}, err
		}
		for _, f := range inside {
			put(File{Src: path.Join(src, f), Target: TargetMods, Rel: path.Join(a.Key, dest, f)})
		}
	}
	return l, validate(l, g)
}
