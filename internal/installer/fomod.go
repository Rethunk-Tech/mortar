package installer

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/Rethunk-Tech/go-fomod"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

var _ = Register(fomodInstaller{})

// fomodInstaller resolves a FOMOD's choices into the files it copies, into the package's folder in the mods folder.
type fomodInstaller struct{}

func (fomodInstaller) ID() string { return "fomod" }

func (fomodInstaller) Detect(a Archive, _ Game) bool {
	p, err := fomod.Find(a.FS)
	return err == nil && p != ""
}

func slash(p string) string { return path.Clean(strings.ReplaceAll(p, `\`, "/")) }

// destination is where op puts src in the package's folder. With no destination the spec installs at the source's own
// path, and an empty one is the package's folder itself.
func destination(op fomod.Op, src string) string {
	switch {
	case op.Destination != nil && *op.Destination != "":
		return *op.Destination
	case op.Destination == nil:
		return src
	}
	return "."
}

// The most operations a config may list and the most files its operations may place before the layout is refused: a
// config is untrusted XML and every folder operation walks its source.
const (
	maxFomodOps   = 20000
	maxFomodFiles = 200000
)

// Counted each time a FOMOD folder operation walks its source. Tests compare a repeated operation to a single one.
var fomodFolderWalks atomic.Int64

// lastOfEach keeps the final occurrence of each identical operation: a later copy of the same operation rewrites what
// the earlier one wrote, so the result is the same with one walk instead of many.
func lastOfEach(ops []fomod.Op) []fomod.Op {
	seen := make(map[string]bool, len(ops))
	kept := make([]fomod.Op, 0, len(ops))
	for _, op := range slices.Backward(ops) {

		dest := "\x00nil"
		if op.Destination != nil {
			dest = *op.Destination
		}
		key := fmt.Sprintf("%q|%q|%t", op.Source, dest, op.Folder)
		if !seen[key] {
			seen[key] = true
			kept = append(kept, op)
		}
	}
	slices.Reverse(kept)
	return kept
}

func (fomodInstaller) Layout(a Archive, g Game, choices Choices) (Layout, error) {
	cfg, cfgPath, ok, err := fomod.Open(a.FS)
	if err != nil || !ok {
		return Layout{}, fmt.Errorf("read the FOMOD config: %w", err)
	}
	// Sources are relative to the folder holding fomod/, which a wrapper folder puts below the archive root.
	wrap := path.Dir(path.Dir(cfgPath))
	base := a
	if wrap != "." {
		var err error
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
		key := winname.Fold(f.Rel)
		if i, seen := at[key]; seen {
			l.Files[i] = f
			return
		}
		at[key] = len(l.Files)
		l.Files = append(l.Files, f)
	}
	ops := fomod.Resolve(cfg, choices, a.Eval)
	if len(ops) > maxFomodOps {
		return Layout{}, fmt.Errorf("%w: the installer lists %d file operations, more than the %d Mortar accepts", ErrUnsafe, len(ops), maxFomodOps)
	}
	listed := map[string][]string{}
	placed := 0
	for _, op := range lastOfEach(ops) {
		src, dest, info, err := fomodOperand(base, op)
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
		inside, ok := listed[src]
		if !ok {
			fomodFolderWalks.Add(1)
			if inside, err = files(base, src); err != nil {
				return Layout{}, err
			}
			listed[src] = inside
		}
		if placed += len(inside); placed > maxFomodFiles {
			return Layout{}, fmt.Errorf("%w: the installer places more than %d files", ErrUnsafe, maxFomodFiles)
		}
		for _, f := range inside {
			put(File{Src: path.Join(src, f), Target: TargetMods, Rel: path.Join(a.Key, dest, f)})
		}
	}
	return l, validate(l, g)
}

// fomodOperand checks an operation's untrusted source and destination and resolves the source in the archive.
func fomodOperand(base Archive, op fomod.Op) (src, dest string, info fs.FileInfo, err error) {
	src = slash(op.Source)
	if !fs.ValidPath(src) {
		return "", "", nil, fmt.Errorf("%w: %q", ErrUnsafe, op.Source)
	}
	dest = destination(op, src)
	// A leading slash means the package's folder, as no slash does.
	dest = slash(cmp.Or(strings.TrimLeft(slash(dest), "/"), "."))
	// A destination is inside the package's own folder: ".." would reach another mod's files.
	if dest == ".." || strings.HasPrefix(dest, "../") {
		return "", "", nil, fmt.Errorf("%w: %q", ErrUnsafe, dest)
	}
	if src, err = safefs.ResolveFold(base.FS, src); err != nil {
		return "", "", nil, err
	}
	info, err = fs.Stat(base.FS, src)
	return src, dest, info, err
}
