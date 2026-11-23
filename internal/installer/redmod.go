package installer

import (
	"encoding/json"
	"io/fs"
	"path"
	"slices"
	"strings"
)

var _ = Register(redmod{})

// redmodSubdirs are the folders a REDmod holds its content in; an info.json beside none of them is not a REDmod.
var redmodSubdirs = []string{"archives", "customSounds", "scripts", "tweaks"}

// redmod lays out a REDmod packed without its mods/ folder (info.json at the top, or in a folder of the top) below
// mods/<name>, where the game's REDmod tool reads it (Vortex's cyberpunk2077_ext_redux installer.redmod.ts: its named
// and top-level layouts). One packed as mods/<name>/info.json is the plain driver's as it stands.
type redmod struct{}

func (redmod) ID() string { return "redmod" }

func (redmod) Detect(a Archive, g Game) bool {
	i := slices.IndexFunc(g.Targets, func(t Target) bool { return t.ID == TargetMods && t.REDmod })
	return i >= 0 && slices.Contains(g.Loaders, "folder") && len(redmodDirs(a)) > 0
}

// redmodDirs are the slash paths of the archive's REDmod folders: the top (".") and its direct subfolders that hold
// an info.json and one of the content folders.
func redmodDirs(a Archive) []string {
	var out []string
	if isRedmodDir(a, ".") {
		out = append(out, ".")
	}
	entries, _ := fs.ReadDir(a.FS, ".")
	for _, e := range entries {
		if e.IsDir() && !skip(e.Name()) && !strings.EqualFold(e.Name(), "mods") && isRedmodDir(a, e.Name()) {
			out = append(out, e.Name())
		}
	}
	return out
}

func isRedmodDir(a Archive, dir string) bool {
	if info, err := fs.Stat(a.FS, path.Join(dir, "info.json")); err != nil || !info.Mode().IsRegular() {
		return false
	}
	return slices.ContainsFunc(redmodSubdirs, func(s string) bool {
		info, err := fs.Stat(a.FS, path.Join(dir, s))
		return err == nil && info.IsDir()
	})
}

func (redmod) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	var l Layout
	for _, dir := range redmodDirs(a) {
		name := redmodName(a, dir)
		all, err := files(a, dir)
		if err != nil {
			return Layout{}, err
		}
		for _, f := range all {
			src := f
			if dir != "." {
				src = path.Join(dir, f)
			}
			if !skip(src) {
				l.Files = append(l.Files, File{Src: src, Target: TargetMods, Rel: path.Join("mods", name, f)})
			}
		}
	}
	return l, validate(l, g)
}

// redmodName is the folder name the REDmod is placed under: the name its info.json gives with dots as underscores,
// as REDmod's tool reads a folder name as the mod's, else the folder it was packed in or the package's key.
func redmodName(a Archive, dir string) string {
	var info struct {
		Name string `json:"name"`
	}
	if b, err := fs.ReadFile(a.FS, path.Join(dir, "info.json")); err == nil {
		_ = json.Unmarshal(b, &info)
	}
	fallback := a.Key
	if dir != "." {
		fallback = dir
	}
	name := strings.ReplaceAll(strings.TrimSpace(info.Name), ".", "_")
	if name == "" || strings.ContainsAny(name, `/\`) {
		return fallback
	}
	return name
}
