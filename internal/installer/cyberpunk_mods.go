package installer

import (
	"encoding/json"
	"path"
	"regexp"
	"slices"
	"strings"
)

func isDLL(f string) bool     { return lower(path.Ext(f)) == ".dll" }
func isReds(f string) bool    { return lower(path.Ext(f)) == ".reds" }
func isArchive(f string) bool { return lower(path.Ext(f)) == ".archive" }
func isXL(f string) bool      { return lower(path.Ext(f)) == ".xl" }
func isArchiveOrXL(f string) bool {
	return isArchive(f) || isXL(f)
}

func withoutREDmods(files []string) []string {
	return slices.DeleteFunc(slices.Clone(files), func(f string) bool { return isUnder(f, redmodBase) })
}

//
// ASI, CET, TweakXL, Audioware
//

func asiInstaller(c *cpCtx) (outcome, bool, error) {
	if len(c.t.filesIn(asiPath, func(f string) bool { return lower(path.Ext(f)) == ".asi" })) == 0 {
		return outcome{}, false, nil
	}
	// Vortex refuses an ASI mod with anything outside bin/x64/plugins; here the rest goes on to the extra files and
	// the as-packed fallback.
	return outcome{kind: "asi", moves: same(c.t.filesUnder(asiPath, nil))}, true, nil
}

func cetDirs(c *cpCtx) []string {
	return c.t.directSubdirsWithSome(cetPrefix, func(f string) bool { return base(f) == "init.lua" })
}

func cetInstaller(c *cpCtx) (outcome, bool, error) {
	if len(cetDirs(c)) == 0 {
		return outcome{}, false, nil
	}
	return outcome{kind: "cet", moves: same(c.t.filesUnder(cetPrefix, nil))}, true, nil
}

func tweakXLInstaller(c *cpCtx) (outcome, bool, error) {
	if !c.t.dirInTree(tweakPrefix) {
		return outcome{}, false, nil
	}
	files := c.t.filesUnder(tweakPrefix, extIs(".yaml", ".yml"))
	if len(files) == 0 {
		return unresolvable(c, "TweakXL")
	}
	return outcome{kind: "tweakxl", moves: same(files)}, true, nil
}

var audiowareExts = extIs(".yaml", ".yml", ".wav", ".ogg", ".mp3", ".flac")

func audiowareInstaller(c *cpCtx) (outcome, bool, error) {
	if !c.t.dirInTree(audiowarePath) {
		return outcome{}, false, nil
	}
	files := c.t.filesUnder(audiowarePath, audiowareExts)
	if len(files) == 0 {
		return unresolvable(c, "Audioware")
	}
	return outcome{kind: "audioware", moves: same(files)}, true, nil
}

//
// Red4Ext
//

var red4extReservedDLLs = []string{
	"clrcompression.dll", "clrjit.dll", "coreclr.dll", "d3dcompiler_47_cor3.dll", "mscordaccore.dll",
	"penimc_cor3.dll", "presentationnative_cor3.dll", "vcruntime140_cor3.dll", "wpfgfx_cor3.dll",
}

func red4extBasedir(c *cpCtx) (outcome, verdict) {
	if !c.t.dirWithSomeIn(red4extBase, isDLL) {
		return outcome{}, noMatch
	}
	return outcome{kind: "red4ext-basedir", moves: moveAll(c.t.filesUnder(red4extBase, nil), red4extBase, red4extBase+"/"+c.name)}, matched
}

func red4extCanonOnly(c *cpCtx) bool {
	return !c.t.dirWithSomeIn(red4extBase, isDLL) && len(c.t.directSubdirsWithSome(red4extBase, isDLL)) > 0
}

func red4extCanon(c *cpCtx) (outcome, verdict) {
	if !red4extCanonOnly(c) {
		return outcome{}, noMatch
	}
	return outcome{kind: "red4ext-canon", moves: same(c.t.filesUnder(red4extBase, nil))}, matched
}

func red4extToplevel(c *cpCtx) (outcome, verdict) {
	if len(c.t.filesIn("", isDLL)) == 0 {
		return outcome{}, noMatch
	}
	return outcome{kind: "red4ext-toplevel", moves: moveAll(c.t.filesUnder("", nil), "", red4extBase+"/"+c.name)}, matched
}

func red4extModnamed(c *cpCtx) (outcome, verdict) {
	dirs := c.t.directSubdirsWithSome("", isDLL)
	switch {
	case len(dirs) == 0:
		return outcome{}, noMatch
	case len(dirs) > 1:
		return outcome{}, conflict
	}
	return outcome{kind: "red4ext-modnamed", moves: moveAll(c.t.filesUnder(dirs[0], nil), "", red4extBase)}, matched
}

func red4extInstaller(c *cpCtx) (outcome, bool, error) {
	topDLLs := c.t.filesIn("", isDLL)
	dllDirs := c.t.allSubdirsWithSome("", isDLL)
	if c.t.pathInTree("red4ext/RED4ext.dll") || (len(dllDirs) == 0 && len(topDLLs) == 0) {
		return outcome{}, false, nil
	}
	var danger []string
	for _, d := range dllDirs {
		if strings.EqualFold(d, "bin/x64") {
			danger = append(danger, d)
		}
	}
	for _, f := range topDLLs {
		if slices.Contains(red4extReservedDLLs, lower(f)) {
			danger = append(danger, f)
		}
	}
	if len(danger) > 0 {
		return outcome{}, false, refuse("Red4Ext Mod Installation Canceled, Dangerous DLL paths: %s", strings.Join(danger, ", "))
	}
	o, v := firstMatching(c, red4extBasedir, red4extCanon, red4extToplevel, red4extModnamed)
	if v != matched {
		return unresolvable(c, "RED4ext")
	}
	return o, true, nil
}

//
// Redscript
//

func redsSideFiles(c *cpCtx) []string {
	return append(c.t.filesIn(redsHints, extIs(".toml")), c.t.filesUnder(redsStorages, nil)...)
}

func redsBasedirDetect(c *cpCtx) bool { return c.t.dirWithSomeIn(redsPrefix, isReds) }
func redsCanonDirs(c *cpCtx) []string { return c.t.topmostSubdirsWithSome(redsPrefix, isReds) }

func redsCanonOnly(c *cpCtx) bool {
	return !redsBasedirDetect(c) && len(redsCanonDirs(c)) > 0
}

func redsToplevelDetect(c *cpCtx) bool {
	return !redsBasedirDetect(c) && !redsCanonOnly(c) && c.t.dirWithSomeIn("", isReds)
}

func redsBasedir(c *cpCtx) (outcome, verdict) {
	if !redsBasedirDetect(c) {
		return outcome{}, noMatch
	}
	moves := moveAll(c.t.filesUnder(redsPrefix, nil), redsPrefix, redsPrefix+"/"+c.name)
	return outcome{kind: "redscript-basedir", moves: append(moves, same(redsSideFiles(c))...)}, matched
}

func redsCanon(c *cpCtx) (outcome, verdict) {
	var files []string
	for _, d := range redsCanonDirs(c) {
		files = append(files, c.t.filesUnder(d, nil)...)
	}
	if len(files) == 0 {
		return outcome{}, noMatch
	}
	return outcome{kind: "redscript-canon", moves: same(files, redsSideFiles(c))}, matched
}

func redsToplevel(c *cpCtx) (outcome, verdict) {
	if !redsToplevelDetect(c) {
		return outcome{}, noMatch
	}
	side := redsSideFiles(c)
	var top []string
	for _, f := range c.t.filesUnder("", nil) {
		if !slices.Contains(side, f) {
			top = append(top, f)
		}
	}
	return outcome{kind: "redscript-toplevel", moves: append(moveAll(top, "", redsPrefix+"/"+c.name), same(side)...)}, matched
}

func redsConfigOnly(c *cpCtx) (outcome, verdict) {
	side := redsSideFiles(c)
	if len(side) == 0 {
		return outcome{}, noMatch
	}
	return outcome{kind: "redscript-config", moves: same(side)}, matched
}

func redscriptInstaller(c *cpCtx) (outcome, bool, error) {
	detected := (len(redsSideFiles(c)) > 0 || c.t.dirWithSomeUnder("", isReds)) && !red4extCanonOnly(c)
	if !detected {
		return outcome{}, false, nil
	}
	o, v := firstMatching(c, redsBasedir, redsCanon, redsToplevel, redsConfigOnly)
	if v != matched {
		return unresolvable(c, "redscript")
	}
	return o, true, nil
}

//
// Archives
//

func archivesCanonXL(c *cpCtx) (outcome, verdict) {
	xls := c.t.filesIn(archivePrefix, isXL)
	if len(xls) == 0 {
		return outcome{}, noMatch
	}
	arch := c.t.filesIn(archivePrefix, isArchive)
	unknown := c.t.filesIn(archivePrefix, func(f string) bool { return !isArchiveOrXL(f) })
	moves := same(xls, arch, unknown)
	if len(withoutREDmods(c.t.filesUnder("", isArchiveOrXL))) != len(moves) {
		return outcome{}, conflict
	}
	return outcome{kind: "archive-xl", moves: moves}, matched
}

func archivesCanon(c *cpCtx) (outcome, verdict) {
	canon := c.t.filesIn(archivePrefix, isArchive)
	if len(canon) == 0 {
		return outcome{}, noMatch
	}
	if len(withoutREDmods(c.t.filesUnder("", isArchiveOrXL))) != len(canon) {
		return outcome{}, conflict
	}
	return outcome{kind: "archive-canon", moves: same(c.t.filesUnder(archivePrefix, nil))}, matched
}

func archivesHeritage(c *cpCtx) (outcome, verdict) {
	if len(c.t.filesIn(heritagePrefix, isArchive)) == 0 {
		return outcome{}, noMatch
	}
	return outcome{kind: "archive-heritage", moves: moveAll(c.t.filesUnder(heritagePrefix, nil), heritagePrefix, archivePrefix)}, matched
}

func archivesOther(c *cpCtx) (outcome, verdict) {
	var files []string
	for _, d := range append(c.rootArchiveDir(), c.t.allSubdirsWithSome("", isArchiveOrXL)...) {
		if d == "." {
			files = append(files, c.t.filesIn("", nil)...)
		} else {
			files = append(files, c.t.filesUnder(d, nil)...)
		}
	}
	moves := make([]move, 0, len(files))
	for _, f := range files {
		if isUnder(f, archivePrefix) {
			moves = append(moves, move{src: f, dst: f})
		} else {
			moves = append(moves, move{src: f, dst: archivePrefix + "/" + f})
		}
	}
	return outcome{kind: "archive-other", moves: moves}, matched
}

// rootArchiveDir is the archive's top as a folder to look in (Vortex's "."), when archives sit loose in it.
func (c *cpCtx) rootArchiveDir() []string {
	if c.t.dirWithSomeIn("", isArchiveOrXL) {
		return []string{"."}
	}
	return nil
}

// archiveReview notes what Vortex warns about when archives landed somewhere a player may need to review.
func (c *cpCtx) archiveReview(moves []move, kind string) {
	var dests []string
	for _, m := range moves {
		dests = append(dests, m.dst)
	}
	d := newFtree(dests)
	subdirs := len(d.subdirNamesIn(archivePrefix)) > 0
	multiple := len(d.filesIn(archivePrefix, isArchive)) > 1 && kind == "archive-other"
	xlInSubdirs := len(d.allSubdirsWithSome(archivePrefix, isXL)) > 0
	var why []string
	if subdirs {
		why = append(why, "archives in subfolders are not read by the game")
	}
	if multiple {
		why = append(why, "more than one top-level archive")
	}
	if xlInSubdirs {
		why = append(why, ".xl files in subfolders are not read by ArchiveXL")
	}
	if len(why) > 0 {
		c.note(NoteArchiveReview, "the archives may need a manual check: "+strings.Join(why, "; "), dests...)
	}
}

func archiveInstaller(c *cpCtx) (outcome, bool, error) {
	if len(c.t.filesIn(archivePrefix, isXL)) == 0 {
		var archives, others []string
		for _, f := range c.t.files {
			if isArchive(f) {
				archives = append(archives, f)
			} else {
				others = append(others, f)
			}
		}
		if len(archives) == 0 {
			return outcome{}, false, nil
		}
		if slices.ContainsFunc(others, func(f string) bool {
			first, _, _ := strings.Cut(path.Dir(f), "/")
			return slices.Contains([]string{"bin", "r6", "red4ext"}, first)
		}) {
			return outcome{}, false, nil
		}
	}
	o, v := firstMatching(c, archivesCanonXL, archivesCanon, archivesHeritage, archivesOther)
	if v != matched {
		return unresolvable(c, "archive")
	}
	c.archiveReview(o.moves, o.kind)
	return o, true, nil
}

// archivesAllowedWithOtherTypes is the canonical archive layouts, which a mod of another type may carry beside its own.
func archivesAllowedWithOtherTypes(c *cpCtx) []move {
	o, v := firstMatching(c, archivesCanonXL, archivesCanon, archivesHeritage)
	if v != matched {
		return nil
	}
	c.archiveReview(o.moves, o.kind)
	return o.moves
}

func archivesToplevel(c *cpCtx) []move {
	files := c.t.filesIn("", isArchiveOrXL)
	if len(files) == 0 {
		return nil
	}
	moves := moveAll(files, "", archivePrefix)
	c.archiveReview(moves, "archive-toplevel")
	return moves
}

//
// Config files
//

var (
	jsonProtected = []string{
		"engine/config/giweights.json", "r6/config/bumpersSettings.json", "r6/config/settings/options.json",
		"r6/config/settings/platform/pc/options.json",
	}
	jsonKnown      = map[string]string{"giweights.json": "engine/config/giweights.json", "bumpersSettings.json": "r6/config/bumpersSettings.json"}
	jsonDirs       = []string{"engine/config", "r6/config", "r6/config/settings", "r6/config/settings/platform/pc"}
	xmlProtected   = []string{"inputContexts.xml", "inputDeadzones.xml", "inputUserMappings.xml", "uiInputActions.xml"}
	jsonProtectedN = []string{"giweights.json", "bumpersSettings.json", "options.json"}
)

func isJSON(f string) bool { return lower(path.Ext(f)) == ".json" }
func isXML(f string) bool  { return lower(path.Ext(f)) == ".xml" }

func protectedBase(names []string) func(string) bool {
	return func(f string) bool {
		return slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, base(f)) })
	}
}

func jsonProtectedFiles(c *cpCtx) []string {
	var out []string
	for _, p := range jsonProtected {
		if c.t.pathInTree(p) {
			out = append(out, c.pathAs(p))
		}
	}
	return out
}

// pathAs is p as the archive spells it.
func (c *cpCtx) pathAs(p string) string {
	if i := slices.IndexFunc(c.t.files, func(f string) bool { return strings.EqualFold(f, p) }); i >= 0 {
		return c.t.files[i]
	}
	return p
}

func jsonToplevel(c *cpCtx) []string {
	return c.t.filesIn("", func(f string) bool { return isJSON(f) && protectedBase(jsonProtectedN)(f) })
}

func jsonInstaller(c *cpCtx) (outcome, bool, error) {
	prot, top := jsonProtectedFiles(c), jsonToplevel(c)
	inDirs := false
	for _, d := range jsonDirs {
		inDirs = inDirs || c.t.dirWithSomeUnder(d, isJSON)
	}
	if len(prot) == 0 && len(top) == 0 && !inDirs {
		return outcome{}, false, nil
	}
	if len(prot) > 0 {
		c.note(NoteProtectedPaths, "this mod writes to config files that other mods or you may have changed", prot...)
		return outcome{kind: "config-json", moves: same(prot)}, true, nil
	}
	if len(top) == 0 || slices.ContainsFunc(top, func(f string) bool { return strings.EqualFold(base(f), "options.json") }) {
		return unresolvable(c, "JSON config")
	}
	var moves []move
	var dests []string
	for _, f := range top {
		for k, v := range jsonKnown {
			if strings.EqualFold(k, base(f)) {
				moves = append(moves, move{src: f, dst: v})
				dests = append(dests, v)
			}
		}
	}
	c.note(NoteProtectedPaths, "this mod writes to config files that other mods or you may have changed", dests...)
	return outcome{kind: "config-json-toplevel", moves: moves}, true, nil
}

func xmlProtectedFiles(c *cpCtx) []string {
	return c.t.filesIn(configXMLBase, func(f string) bool { return isXML(f) && protectedBase(xmlProtected)(f) })
}

func xmlCanonFiles(c *cpCtx) []string {
	return c.t.filesIn(configXMLBase, func(f string) bool { return isXML(f) && !protectedBase(xmlProtected)(f) })
}

func xmlMergeable(c *cpCtx) []string { return c.t.filesIn(mergeableBase, isXML) }

func xmlToplevel(c *cpCtx) []string {
	return c.t.filesIn("", func(f string) bool { return isXML(f) && protectedBase(xmlProtected)(f) })
}

func xmlLayouts(c *cpCtx, withToplevel bool) (outcome, verdict) {
	if p := xmlProtectedFiles(c); len(p) > 0 {
		return outcome{kind: "config-xml-protected", moves: same(p, xmlCanonFiles(c))}, matched
	}
	if cn := xmlCanonFiles(c); len(cn) > 0 {
		return outcome{kind: "config-xml", moves: same(cn)}, matched
	}
	if m := xmlMergeable(c); len(m) > 0 {
		return outcome{kind: "config-xml-mergeable", moves: same(m)}, matched
	}
	if t := xmlToplevel(c); withToplevel && len(t) > 0 {
		return outcome{kind: "config-xml-toplevel", moves: moveAll(t, "", configXMLBase)}, matched
	}
	return outcome{}, noMatch
}

func xmlInstaller(c *cpCtx) (outcome, bool, error) {
	if len(xmlProtectedFiles(c))+len(xmlCanonFiles(c))+len(xmlMergeable(c))+len(xmlToplevel(c)) == 0 {
		return outcome{}, false, nil
	}
	o, v := xmlLayouts(c, true)
	if v != matched {
		return unresolvable(c, "XML config")
	}
	if o.kind == "config-xml-protected" || o.kind == "config-xml-toplevel" {
		var dests []string
		for _, m := range o.moves {
			if protectedBase(xmlProtected)(m.dst) {
				dests = append(dests, m.dst)
			}
		}
		c.note(NoteProtectedPaths, "this mod writes to config files that other mods or you may have changed", dests...)
	}
	return o, true, nil
}

// iniInstaller places .ini files: ReShade presets in bin/x64 with their shaders, the rest in engine/config/platform/pc.
func iniInstaller(c *cpCtx) (outcome, bool, error) {
	inis := c.t.filesUnder("", extIs(".ini"))
	if len(inis) == 0 {
		return outcome{}, false, nil
	}
	if slices.ContainsFunc(c.t.files, func(f string) bool { return strings.Contains(base(f), "init.lua") || isReds(f) }) || c.t.pathInTree("bin/x64/global.ini") {
		return outcome{}, false, nil
	}
	reshade := false
	if b, err := c.read(inis[0]); err == nil {
		reshade = !iniHeader.Match(b)
	}
	var moves []move
	for _, f := range inis {
		dir := "engine/config/platform/pc"
		if reshade {
			dir = "bin/x64"
		}
		moves = append(moves, move{src: f, dst: dir + "/" + base(f)})
	}
	if reshade {
		for _, f := range c.t.files {
			if _, tail, ok := strings.CutLast(f, "reshade-shaders"); ok {
				moves = append(moves, move{src: f, dst: "bin/x64/reshade-shaders" + tail})
			}
		}
	}
	return outcome{kind: "ini", moves: moves}, true, nil
}

var iniHeader = regexp.MustCompile(`^[\[#].+`)

//
// Presets
//

var (
	presetCyberCatKeys = []string{"DataExists", "Unknown1", "UnknownFirstBytes", "FirstSection", "SecondSection", "ThirdSection", "StringTriples"}
	presetFemale       = regexp.MustCompile(`LocKey#14444638123505366956:\d+`)
	presetMale         = regexp.MustCompile(`LocKey#\d+:\d+`)
)

func isPreset(f string) bool { return lower(path.Ext(f)) == ".preset" }

// presetDir is where a preset goes by what it holds, "" for a file that is no preset.
func presetDir(content []byte) string {
	var obj map[string]json.RawMessage
	if json.Unmarshal(content, &obj) == nil {
		if !slices.ContainsFunc(presetCyberCatKeys, func(k string) bool { _, ok := obj[k]; return !ok }) {
			return cybercatPreset
		}
	} else if presetFemale.Match(content) {
		return unlockerPreset + "/female"
	}
	if presetMale.Match(content) {
		return unlockerPreset + "/male"
	}
	return ""
}

func presetByContent(c *cpCtx, kind, dir string) (outcome, verdict) {
	candidates := c.t.filesIn(dir, isPreset)
	var moves []move
	for _, f := range candidates {
		b, err := c.read(f)
		if err != nil {
			continue
		}
		if d := presetDir(b); d != "" {
			moves = append(moves, move{src: f, dst: d + "/" + base(f)})
		}
	}
	switch {
	case len(moves) == 0:
		return outcome{}, noMatch
	case len(moves) != len(candidates):
		return outcome{}, conflict
	}
	return outcome{kind: kind, moves: moves}, matched
}

func presetInstaller(c *cpCtx) (outcome, bool, error) {
	if !c.t.dirWithSomeUnder("", isPreset) {
		return outcome{}, false, nil
	}
	o, v := firstMatching(c,
		func(c *cpCtx) (outcome, verdict) {
			if f := c.t.filesIn(cybercatPreset, isPreset); len(f) > 0 {
				return outcome{kind: "preset-cybercat", moves: same(f)}, matched
			}
			return outcome{}, noMatch
		},
		func(c *cpCtx) (outcome, verdict) {
			f := append(c.t.filesIn(unlockerPreset+"/female", isPreset), c.t.filesIn(unlockerPreset+"/male", isPreset)...)
			if len(f) > 0 {
				return outcome{kind: "preset-unlocker", moves: same(f)}, matched
			}
			return outcome{}, noMatch
		},
		func(c *cpCtx) (outcome, verdict) { return presetByContent(c, "preset-unlocker-legacy", unlockerPreset) },
		func(c *cpCtx) (outcome, verdict) { return presetByContent(c, "preset-toplevel", "") },
	)
	if v != matched {
		return unresolvable(c, "character preset")
	}
	return o, true, nil
}
