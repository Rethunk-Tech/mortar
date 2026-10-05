package installer

import (
	"context"
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

func (fomodInstaller) Layout(_ context.Context, a Archive, g Game, choices Choices) (Layout, error) {
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
	for _, op := range fomod.Resolve(cfg, choices, fomod.EvalContext{}) {
		src := slash(op.Source)
		if !fs.ValidPath(src) {
			return Layout{}, fmt.Errorf("%w: %q", ErrUnsafe, op.Source)
		}
		dest := op.Destination
		if dest == "" {
			dest = path.Base(src)
		}
		dest = slash(dest)
		info, err := fs.Stat(a.FS, src)
		if err != nil {
			return Layout{}, err
		}
		if !info.IsDir() && !op.Folder {
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
