package pack

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// LocalMod is a mod the other manager added from disk rather than from a site, so its files come with the pack.
// Dir is its plugin folder on this computer; otherwise Files are what a code carried, relative to that folder.
type LocalMod struct {
	Name, Author, Version string
	Disabled              bool
	Dir                   string
	Files                 []File
}

// offSuffix is what r2modman and Gale add to every file of a switched-off mod (Kesomannen/gale
// src-tauri/src/profile/install/fs.rs toggle_file), possibly more than once.
const offSuffix = ".old"

func switchedOn(name string) string {
	for strings.HasSuffix(name, offSuffix) {
		name = strings.TrimSuffix(name, offSuffix)
	}
	return name
}

// Size is the bytes the mod's files take.
func (m LocalMod) Size() int64 {
	if m.Dir != "" {
		n, _ := datadir.Size(m.Dir)
		return n
	}
	var n int64
	for _, f := range m.Files {
		n += int64(len(f.Data))
	}
	return n
}

// Stage writes the mod's files below dir as BepInEx/plugins/..., the layout the BepInEx installer places as one
// plugin folder, under their switched-on names. Links in Dir are not followed.
func (m LocalMod) Stage(dir string) error {
	root := filepath.Join(dir, "BepInEx", "plugins")
	dest := func(rel string) (string, error) {
		p := filepath.Join(root, filepath.FromSlash(path.Join(path.Dir(rel), switchedOn(path.Base(rel)))))
		return p, os.MkdirAll(filepath.Dir(p), 0o700)
	}
	if m.Dir == "" {
		for _, f := range m.Files {
			p, err := dest(f.Path)
			if err == nil {
				err = fsx.WriteFile(p, f.Data, 0o600)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	return filepath.WalkDir(m.Dir, func(src string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(m.Dir, src)
		if err != nil {
			return err
		}
		p, err := dest(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		return datadir.CopyFile(src, p)
	})
}

// carriedPlugins takes out of loose each folder below BepInEx/plugins that no package of the pack names and that holds
// a plugin DLL: a mod the sender added from disk, whose DLLs are the one executable type a pack may bring. Any other
// executable stays in loose, where ProfileFiles refuses it. A folder whose every file is switched off is a disabled mod.
func carriedPlugins(loose []File, packages []Ref) (local []LocalMod, rest []File) {
	groups := map[string]*LocalMod{}
	var order []string
	for _, f := range loose {
		w, ok := winPath(f.Path)
		segs := strings.SplitN(w, "/", 4)
		if !ok || len(segs) < 4 || !strings.EqualFold(segs[0], "BepInEx") || !strings.EqualFold(segs[1], "plugins") ||
			slices.ContainsFunc(packages, func(r Ref) bool { return strings.EqualFold(r.Native, segs[2]) }) ||
			(blocked(switchedOn(w)) && !strings.HasSuffix(strings.ToLower(switchedOn(w)), ".dll")) {
			rest = append(rest, f)
			continue
		}
		g, seen := groups[segs[2]]
		if !seen {
			g = &LocalMod{Name: segs[2], Disabled: true}
			groups[segs[2]] = g
			order = append(order, segs[2])
		}
		g.Files = append(g.Files, File{Path: segs[3], Data: f.Data})
		g.Disabled = g.Disabled && strings.HasSuffix(segs[3], offSuffix)
	}
	for _, name := range order {
		g := groups[name]
		if slices.ContainsFunc(g.Files, func(f File) bool { return strings.HasSuffix(strings.ToLower(switchedOn(f.Path)), ".dll") }) {
			local = append(local, *g)
			continue
		}
		for _, f := range g.Files {
			rest = append(rest, File{Path: "BepInEx/plugins/" + name + "/" + f.Path, Data: f.Data})
		}
	}
	slices.SortFunc(rest, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return local, rest
}
