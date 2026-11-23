package installer

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

// REDmods (Vortex's installer.redmod.ts): a folder holding an info.json and one or more of archives/, customSounds/,
// scripts/ and tweaks/. It is laid out as mods/<name>, where REDmod's tool reads it, whether it was packed that way,
// in a folder of its own, or at the archive's top.

const (
	redmodInfoFile = "info.json"
	redmodMissing  = "REDmods require at least one mod file or placeholder in addition to an info.json"
)

var (
	redmodSubdirs       = []string{"archives", "customSounds", "scripts", "tweaks"}
	redmodScriptSubdirs = []string{"core", "cyberpunk", "exec", "samples", "tests"}
	redmodTweakSubdirs  = []string{"base/gameplay/static_data", "ep1/gameplay/static_data"}
)

type redmodInfo struct {
	Name         string `json:"name"`
	CustomSounds []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"customSounds"`
}

func isInfoJSON(f string) bool { return base(f) == redmodInfoFile }

func hasRedmodSubdir(c *cpCtx, dir string) bool {
	return slices.ContainsFunc(c.t.subdirNamesIn(dir), func(s string) bool {
		return slices.ContainsFunc(redmodSubdirs, func(r string) bool { return strings.EqualFold(r, s) })
	})
}

func redmodNamedDirs(c *cpCtx) []string {
	return slices.DeleteFunc(c.t.directSubdirsWithSome("", isInfoJSON), func(d string) bool { return !hasRedmodSubdir(c, d) })
}

func redmodToplevel(c *cpCtx) bool {
	return c.t.dirWithSomeIn("", isInfoJSON) && hasRedmodSubdir(c, "")
}

func redmodInstaller(c *cpCtx) (outcome, bool, error) {
	canon, named, top := c.t.dirInTree(redmodBase), len(redmodNamedDirs(c)) > 0, redmodToplevel(c)
	if !canon && !named && !top {
		return outcome{}, false, nil
	}
	var dirs []string
	switch {
	case canon:
		var err error
		if dirs, err = redmodCanonDirs(c); err != nil {
			return outcome{}, false, err
		}
	case named:
		dirs = redmodNamedDirs(c)
	default:
		dirs = []string{""}
	}
	moves, err := redmodMoves(c, dirs)
	if err != nil {
		return outcome{}, false, err
	}
	return outcome{kind: "redmod", moves: moves}, true, nil
}

// redmodWithOtherTypes is the REDmods a mod of several types carries, which only the canonical layout may hold.
func redmodWithOtherTypes(c *cpCtx) ([]move, error) {
	if !c.t.dirInTree(redmodBase) {
		return nil, nil
	}
	dirs, err := redmodCanonDirs(c)
	if err != nil {
		return nil, err
	}
	return redmodMoves(c, dirs)
}

func redmodCanonDirs(c *cpCtx) ([]string, error) {
	valid := c.t.directSubdirsWithSome(redmodBase, isInfoJSON)
	var invalid []string
	for _, n := range c.t.subdirNamesIn(redmodBase) {
		d := redmodBase + "/" + n
		if !slices.ContainsFunc(valid, func(v string) bool { return strings.EqualFold(v, d) }) {
			invalid = append(invalid, d)
		}
	}
	if len(invalid) > 0 {
		return nil, refuse("REDmod: these directories don't look like valid REDmods: %s", strings.Join(invalid, ", "))
	}
	return valid, nil
}

func redmodMoves(c *cpCtx, dirs []string) ([]move, error) {
	var out []move
	for _, dir := range dirs {
		ms, err := redmodOne(c, dir)
		if err != nil {
			return nil, err
		}
		out = append(out, ms...)
	}
	return out, nil
}

func redmodOne(c *cpCtx, src string) ([]move, error) {
	raw, err := c.read(path.Join(src, redmodInfoFile))
	var info redmodInfo
	if err == nil {
		var probe struct {
			Version *string `json:"version"`
		}
		if err = json.Unmarshal(raw, &info); err == nil {
			err = json.Unmarshal(raw, &probe)
		}
		if err == nil && (info.Name == "" || probe.Version == nil) {
			err = fmt.Errorf("it needs a name and a version")
		}
	}
	if err != nil {
		return nil, refuse("REDmod: error decoding %s: %v", path.Join(src, redmodInfoFile), err)
	}
	dst := path.Join(redmodBase, strings.ReplaceAll(info.Name, ".", "_"))
	var out []move
	for _, step := range []func(*cpCtx, string, string, redmodInfo) ([]move, error){redmodArchives, redmodSounds, redmodScripts, redmodTweaks} {
		ms, err := step(c, src, dst, info)
		if err != nil {
			return nil, err
		}
		out = append(out, ms...)
	}
	out = append(out, move{src: path.Join(src, redmodInfoFile), dst: path.Join(dst, redmodInfoFile)})
	out = append(out, redmodExtras(c, src, dst)...)
	dests := newFtree(destinations(out))
	hasSub := slices.ContainsFunc(dests.subdirNamesIn(dst), func(s string) bool { return slices.Contains(redmodSubdirs, s) })
	if !hasSub && !redmodSkipOnly(info) {
		return nil, refuse("REDmod: %s", redmodMissing)
	}
	return out, nil
}

func destinations(ms []move) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.dst
	}
	return out
}

func redmodSkipOnly(info redmodInfo) bool {
	if len(info.CustomSounds) == 0 {
		return false
	}
	return !slices.ContainsFunc(info.CustomSounds, func(s struct {
		Name string `json:"name"`
		Type string `json:"type"`
	},
	) bool {
		return s.Type != "mod_skip"
	})
}

func fromTo(files []string, src, dst string) []move { return moveAll(files, src, dst) }

func redmodArchives(c *cpCtx, src, dst string, _ redmodInfo) ([]move, error) {
	dir := path.Join(src, "archives")
	proper := c.t.filesIn(dir, isArchiveOrXL)
	all := c.t.filesUnder(dir, isArchiveOrXL)
	hasXL := slices.ContainsFunc(proper, isXL)
	if len(all) != len(proper) || (len(proper) > 1 && !hasXL) {
		c.note(NoteREDmodSublayout, "the archives in this REDmod may need a manual check: more than one archive, or archives in subfolders", all...)
	}
	return fromTo(all, src, dst), nil
}

func redmodSounds(c *cpCtx, src, dst string, info redmodInfo) ([]move, error) {
	files := c.t.filesUnder(path.Join(src, "customSounds"), extIs(".wav"))
	skips := 0
	for _, s := range info.CustomSounds {
		if s.Type == "mod_skip" {
			skips++
		}
	}
	requires := skips != len(info.CustomSounds)
	if (requires && len(files) == 0) || (!requires && len(files) > 0) {
		return nil, refuse("REDmod: customSounds sublayout: sound files required %v, present %v", requires, len(files) > 0)
	}
	return fromTo(files, src, dst), nil
}

func redmodScripts(c *cpCtx, src, dst string, _ redmodInfo) ([]move, error) {
	dir := path.Join(src, "scripts")
	all := c.t.filesUnder(dir, extIs(".script", ".ws"))
	var valid int
	for _, s := range redmodScriptSubdirs {
		valid += len(c.t.filesUnder(path.Join(dir, s), extIs(".script", ".ws")))
	}
	if valid != len(all) {
		return nil, refuse("REDmod: scripts must sit in one of %s: %d of %d files do not", strings.Join(redmodScriptSubdirs, ", "), len(all)-valid, len(all))
	}
	return fromTo(all, src, dst), nil
}

func redmodTweaks(c *cpCtx, src, dst string, _ redmodInfo) ([]move, error) {
	dir := path.Join(src, "tweaks")
	all := c.t.filesUnder(dir, extIs(".tweak"))
	var valid int
	for _, s := range redmodTweakSubdirs {
		valid += len(c.t.filesUnder(path.Join(dir, s), extIs(".tweak")))
	}
	if valid != len(all) {
		return nil, refuse("REDmod: tweaks must sit in %s: %d of %d files do not", strings.Join(redmodTweakSubdirs, " or "), len(all)-valid, len(all))
	}
	return fromTo(all, src, dst), nil
}

// redmodExtras are the REDmod folder's other files, which go along.
func redmodExtras(c *cpCtx, src, dst string) []move {
	var files []string
	files = append(files, c.t.filesIn(src, func(f string) bool { return !isInfoJSON(f) })...)
	for _, n := range c.t.subdirNamesIn(src) {
		if !slices.ContainsFunc(redmodSubdirs, func(r string) bool { return strings.EqualFold(r, n) }) {
			files = append(files, c.t.filesUnder(path.Join(src, n), nil)...)
		}
	}
	return fromTo(files, src, dst)
}
