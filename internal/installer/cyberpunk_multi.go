package installer

import (
	"encoding/json"
	"path"
	"regexp"
	"slices"
)

// AMM (Appearance Menu Mod) add-ons (Vortex's installer.amm.ts): Lua and JSON files AMM reads from its own folders,
// found by where they are packed or, for loose files, by what they hold.

var (
	ammLua = []struct {
		dir      string
		required []*regexp.Regexp
	}{
		{ammBase + "/Collabs/Custom Appearances", rx(`modder\s*=`, `unique_identifier\s*=`, `entity_id\s*=`, `appearances\s*=`)},
		{ammBase + "/Collabs/Custom Entities", rx(`modder\s*=`, `unique_identifier\s*=`, `entity_info\s*=`)},
		{ammBase + "/Collabs/Custom Poses", rx(`modder\s*=`, `category\s*=`, `entity_path\s*=`, `anims\s*=`)},
		{ammBase + "/Collabs/Custom Props", rx(`modder\s*=`, `unique_identifier\s*=`, `props\s*=`)},
	}
	ammJSON = []struct {
		dir  string
		keys []string
	}{
		{ammBase + "/User/Decor", []string{"name", "props", "lights"}},
		{ammBase + "/User/Locations", []string{"x", "y", "z"}},
		{ammBase + "/User/Scripts", []string{"title", "actors"}},
		{ammBase + "/User/Themes", []string{"Text", "Border"}},
	}
)

func rx(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		out[i] = regexp.MustCompile(p)
	}
	return out
}

func isLua(f string) bool { return lower(path.Ext(f)) == ".lua" }

func ammFiles(c *cpCtx, dir string, pred func(string) bool) []string {
	var out []string
	for _, d := range c.t.allSubdirsWithSome(dir, pred) {
		for _, f := range c.t.filesUnder(d, nil) {
			if !slices.Contains(out, f) {
				out = append(out, f)
			}
		}
	}
	return out
}

func ammCanonFiles(c *cpCtx) []string {
	return append(ammFiles(c, ammBase+"/Collabs", isLua), ammFiles(c, ammBase+"/User", isJSON)...)
}

func ammToplevelSubdirFiles(c *cpCtx) []string {
	return append(ammFiles(c, "Collabs", isLua), ammFiles(c, "User", isJSON)...)
}

func ammCanon(c *cpCtx) (outcome, verdict) {
	files := ammCanonFiles(c)
	if len(files) == 0 {
		return outcome{}, noMatch
	}
	if len(files) != len(c.t.filesUnder(ammBase, nil)) {
		return outcome{}, conflict
	}
	return outcome{kind: "amm", moves: append(same(files), archivesAllowedWithOtherTypes(c)...)}, matched
}

func ammToplevelSubdir(c *cpCtx) (outcome, verdict) {
	files := ammToplevelSubdirFiles(c)
	if len(files) == 0 {
		return outcome{}, noMatch
	}
	return outcome{kind: "amm-toplevel-subdir", moves: append(moveAll(files, "", ammBase), archivesToplevel(c)...)}, matched
}

func ammDirFor(content []byte, ext string) string {
	if ext == ".json" {
		var obj map[string]json.RawMessage
		if json.Unmarshal(content, &obj) != nil {
			return ""
		}
		for _, k := range ammJSON {
			if len(obj) >= len(k.keys) && !slices.ContainsFunc(k.keys, func(key string) bool { _, ok := obj[key]; return !ok }) {
				return k.dir
			}
		}
		return ""
	}
	for _, k := range ammLua {
		if !slices.ContainsFunc(k.required, func(r *regexp.Regexp) bool { return !r.Match(content) }) {
			return k.dir
		}
	}
	return ""
}

func ammToplevel(c *cpCtx) (outcome, verdict) {
	candidates := c.t.filesIn("", func(f string) bool { return isLua(f) || isJSON(f) })
	var moves []move
	for _, f := range candidates {
		b, err := c.read(f)
		if err != nil {
			continue
		}
		if d := ammDirFor(b, lower(path.Ext(f))); d != "" {
			moves = append(moves, move{src: f, dst: d + "/" + base(f)})
		}
	}
	switch {
	case len(moves) == 0:
		return outcome{}, noMatch
	case len(moves) != len(candidates):
		return outcome{}, conflict
	}
	return outcome{kind: "amm-toplevel", moves: append(moves, archivesToplevel(c)...)}, matched
}

func ammInstaller(c *cpCtx) (outcome, bool, error) {
	looksLike := c.t.dirInTree(ammBase) || len(c.t.directSubdirsWithSome("Collabs", isLua)) > 0 || len(c.t.directSubdirsWithSome("User", isJSON)) > 0
	if !looksLike {
		if len(c.t.filesIn("", func(f string) bool { return isLua(f) || isJSON(f) })) == 0 {
			return outcome{}, false, nil
		}
		if _, v := ammToplevel(c); v != matched {
			return outcome{}, false, nil
		}
	}
	o, v := firstMatching(c, ammCanon, ammToplevelSubdir)
	if v != matched {
		o, v = ammToplevel(c)
	}
	if v != matched {
		return unresolvable(c, "Appearance Menu Mod")
	}
	return o, true, nil
}

// multiType is a mod that carries more than one of the kinds above (Vortex's installer.multitype.ts): each part is
// laid out as its own kind would, and files outside every part make it unresolvable.
func multiType(c *cpCtx) (outcome, bool, error) {
	canonXML := func() bool {
		return len(xmlProtectedFiles(c))+len(xmlCanonFiles(c))+len(xmlMergeable(c)) > 0
	}
	kinds := []bool{
		len(c.t.filesIn(archivePrefix, isXL)) > 0 || len(c.t.filesIn(archivePrefix, isArchive)) > 0 || len(c.t.filesIn(heritagePrefix, isArchive)) > 0,
		c.t.dirInTree(audiowarePath),
		len(jsonProtectedFiles(c)) > 0,
		canonXML(),
		len(cetDirs(c)) > 0,
		c.t.dirInTree(redmodBase),
		redsBasedirDetect(c) || redsCanonOnly(c) || len(redsSideFiles(c)) > 0,
		red4extCanonOnly(c),
		c.t.dirWithSomeIn(red4extBase, isDLL),
		c.t.dirInTree(tweakPrefix),
	}
	n := 0
	for _, k := range kinds {
		if k {
			n++
		}
	}
	if n < 2 {
		return outcome{}, false, nil
	}
	var moves []move
	if p := jsonProtectedFiles(c); len(p) > 0 {
		c.note(NoteProtectedPaths, "this mod writes to config files that other mods or you may have changed", p...)
		moves = append(moves, same(p)...)
	}
	if x, v := xmlLayouts(c, false); v == matched {
		if x.kind == "config-xml-protected" {
			c.note(NoteProtectedPaths, "this mod writes to config files that other mods or you may have changed", xmlProtectedFiles(c)...)
		}
		moves = append(moves, x.moves...)
	}
	if len(cetDirs(c)) > 0 {
		moves = append(moves, same(c.t.filesUnder(cetPrefix, nil))...)
	}
	for _, l := range []layoutFn{red4extBasedir, red4extCanon} {
		if o, v := l(c); v == matched {
			moves = append(moves, o.moves...)
		}
	}
	if c.t.dirInTree(tweakPrefix) {
		moves = append(moves, same(c.t.filesUnder(tweakPrefix, extIs(".yaml", ".yml")))...)
	}
	if c.t.dirInTree(audiowarePath) {
		moves = append(moves, same(c.t.filesUnder(audiowarePath, audiowareExts))...)
	}
	if o, v := firstMatching(c, redsBasedir, redsCanon, redsConfigOnly); v == matched {
		moves = append(moves, o.moves...)
	}
	moves = append(moves, archivesAllowedWithOtherTypes(c)...)
	rm, err := redmodWithOtherTypes(c)
	if err != nil {
		return outcome{}, false, refuse("a mod with several parts cannot be installed when its REDmod part fails: %v", err)
	}
	moves = append(moves, rm...)
	if len(moves) == 0 {
		return unresolvable(c, "multi-part")
	}
	return outcome{kind: "multitype", moves: moves}, true, nil
}
