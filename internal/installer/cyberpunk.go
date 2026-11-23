package installer

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// The Cyberpunk 2077 installer is Vortex's cyberpunk2077_ext_redux installer pipeline (installers.ts, installers.layouts.ts
// and installer.*.ts there) as one driver: the same archive shapes in the same order, the same places mis-packed
// archives are moved to, and the same refusals. Where Vortex asks the player a question, the layout carries a Note
// instead (Problems shows it), and an archive no shape takes is laid out as packed.

// DriverCyberpunk is the id of the driver the mods target's `layout: "cyberpunk"` selects.
const DriverCyberpunk = "cyberpunk"

// LayoutCyberpunk is the catalog's name for this driver (components.TargetDef.Layout).
const LayoutCyberpunk = components.LayoutCyberpunk

// Note codes of what the Cyberpunk installer saw in an archive.
const (
	NoteDeprecatedCore  = "deprecatedCore"
	NoteProtectedPaths  = "protectedPaths"
	NoteUnrecognised    = "unrecognisedLayout"
	NoteArchiveReview   = "archiveReview"
	NoteExtraFiles      = "extraFilesMoved"
	NoteREDmodSublayout = "redmodArchiveReview"
)

const (
	archivePrefix  = "archive/pc/mod"
	heritagePrefix = "archive/pc/patch"
	cetPrefix      = "bin/x64/plugins/cyber_engine_tweaks/mods"
	redsPrefix     = "r6/scripts"
	redsHints      = "r6/config/redsUserHints"
	redsStorages   = "r6/storages"
	red4extBase    = "red4ext/plugins"
	tweakPrefix    = "r6/tweaks"
	audiowarePath  = "r6/audioware"
	asiPath        = "bin/x64/plugins"
	configXMLBase  = "r6/config"
	mergeableBase  = "r6/input"
	ammBase        = cetPrefix + "/AppearanceMenuMod"
	// ExtraFilesBase is where documentation and pictures an archive carries are put, below a folder named for the mod.
	ExtraFilesBase = "V2077/mod-extra-files"
	cybercatBase   = "CyberCAT"
	cybercatPreset = "V2077/presets/cybercat"
	unlockerPreset = cetPrefix + "/AppearanceChangeUnlocker/character-presets"
	redmodBase     = "mods"
	redmodCache    = "r6/cache/modded"
)

var knownToplevelDirs = []string{"archive", "bin", "engine", "r6", "red4ext", "mods"}

var extraFileExts = []string{".md", ".txt", ".pdf", ".png", ".jpg", ".webp", ".gif", ".svg", ".odt", ".rtf", ".doc"}

// move is a file going from src in the archive to dst below the game folder.
type move struct{ src, dst string }

type verdict int

const (
	noMatch verdict = iota
	matched
	conflict
)

type outcome struct {
	kind  string
	moves []move
}

func same(files ...[]string) []move {
	var out []move
	for _, fs := range files {
		for _, f := range fs {
			out = append(out, move{src: f, dst: f})
		}
	}
	return out
}

func moveAll(files []string, from, to string) []move {
	out := make([]move, len(files))
	for i, f := range files {
		out[i] = move{src: f, dst: replacePrefix(f, from, to)}
	}
	return out
}

// cpCtx is one archive being laid out: its files with any wrapping folder taken off, the name its mod goes by where a
// folder is named for it, and what the archive's files hold where a layout depends on it.
type cpCtx struct {
	t     ftree
	name  string
	read  func(rel string) ([]byte, error)
	notes []Note
}

func (c *cpCtx) note(code, msg string, files ...string) {
	n := Note{Code: code, Message: msg}
	if len(files) > 0 {
		n.Files = files
	}
	c.notes = append(c.notes, n)
}

type layoutFn func(c *cpCtx) (outcome, verdict)

func firstMatching(c *cpCtx, layouts ...layoutFn) (outcome, verdict) {
	for _, l := range layouts {
		if o, v := l(c); v != noMatch {
			return o, v
		}
	}
	return outcome{}, noMatch
}

// Detect reports whether the game's mods target is the Cyberpunk layout.
func (cyberpunk) Detect(_ Archive, g Game) bool {
	return slices.Contains(g.Loaders, "folder") && slices.ContainsFunc(g.Targets, func(t Target) bool { return t.ID == TargetMods && t.Layout == LayoutCyberpunk })
}

type cyberpunk struct{}

var _ = Register(cyberpunk{})

func (cyberpunk) ID() string { return DriverCyberpunk }

// errRefused is an archive the installer will not lay out, for a reason worded for the player.
var errRefused = errors.New("refused")

func refuse(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errRefused, fmt.Sprintf(format, args...))
}

func (cyberpunk) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	all, err := files(a, ".")
	if err != nil {
		return Layout{}, err
	}
	all = slices.DeleteFunc(all, skip)
	wrapper, inner := unwrap(all)
	c := &cpCtx{t: newFtree(inner), name: modName(a, wrapper, inner), read: func(rel string) ([]byte, error) {
		return fs.ReadFile(a.FS, path.Join(wrapper, rel))
	}}
	l, err := c.layout()
	if err != nil {
		return Layout{}, err
	}
	for i := range l.Files {
		l.Files[i].Src = path.Join(wrapper, l.Files[i].Src)
	}
	return l, validate(l, g)
}

// unwrap takes off the one folder an archive is wrapped in when it holds a game folder (Vortex's giftwrap layout).
func unwrap(all []string) (wrapper string, inner []string) {
	top := map[string]bool{}
	for _, f := range all {
		first, _, ok := strings.Cut(f, "/")
		if !ok {
			return "", all
		}
		top[first] = true
	}
	if len(top) != 1 {
		return "", all
	}
	var dir string
	for d := range top {
		dir = d
	}
	t := newFtree(all)
	if !slices.ContainsFunc(t.subdirNamesIn(dir), func(s string) bool { return slices.Contains(knownToplevelDirs, s) }) {
		return "", all
	}
	for _, f := range all {
		inner = append(inner, strings.TrimPrefix(f, dir+"/"))
	}
	return dir, inner
}

// modName is the name a layout that needs one gives the mod: the archive's own (Archive.Name), else the folder it was
// wrapped in, else a loose file's name, else its store key. Vortex takes it from the download's file name, which a
// stored archive no longer has.
func modName(a Archive, wrapper string, inner []string) string {
	for _, n := range []string{a.Name, wrapper} {
		if n = cleanName(n); n != "" {
			return n
		}
	}
	top := slices.IndexFunc(inner, func(f string) bool { return !strings.Contains(f, "/") })
	if top >= 0 {
		if n := cleanName(strings.TrimSuffix(inner[top], path.Ext(inner[top]))); n != "" {
			return n
		}
	}
	return cleanName(a.Key)
}

func cleanName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 0x20 {
			return '_'
		}
		return r
	}, s)
	return strings.Trim(strings.TrimSpace(s), ".")
}

// layout runs the pipeline: the first shape that takes the archive lays it out, then files it left behind that are
// documentation go to the extra files folder, and an archive still not whole is laid out as packed.
func (c *cpCtx) layout() (Layout, error) {
	o, err := c.pick()
	if err != nil {
		return Layout{}, err
	}
	moves := dedupe(o.moves)
	total := len(c.t.files)
	if o.kind == "redmod" || o.kind == "multitype" {
		// Mods ship a placeholder in the folder REDmod's tool generates, which Mortar makes itself.
		total -= len(c.t.filesUnder(redmodCache, nil))
	}
	if len(moves) < total {
		extra := c.extraFiles()
		moves = dedupe(append(moves, extra...))
		if len(extra) > 0 {
			c.note(NoteExtraFiles, "documentation and pictures in this mod are in "+ExtraFilesBase+"/"+c.name)
		}
	}
	if len(moves) < total {
		c.note(NoteUnrecognised, "this mod does not match a layout Mortar knows, so every file is placed as packed", c.t.files...)
		o = outcome{kind: "fallback"}
		moves = same(c.t.files)
	}
	l := Layout{Kind: o.kind, Notes: c.notes, Unrecognised: o.kind == "fallback" || o.kind == "unresolvable"}
	sort.Slice(moves, func(i, j int) bool {
		return cmp.Or(cmp.Compare(moves[i].dst, moves[j].dst), cmp.Compare(moves[i].src, moves[j].src)) < 0
	})
	for _, m := range moves {
		l.Files = append(l.Files, File{Src: m.src, Target: TargetMods, Rel: m.dst})
	}
	return l, nil
}

func dedupe(in []move) []move {
	seen := map[move]bool{}
	var out []move
	for _, m := range in {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// pick is Vortex's installer pipeline (installers.ts) in its order; an installer that cannot make sense of an archive
// it recognised falls back to placing it as packed (Vortex asks first, see unresolvable).
func (c *cpCtx) pick() (outcome, error) {
	steps := []func(*cpCtx) (outcome, bool, error){
		coreCET, coreRedscript, coreRed4ext, coreDeprecatedTools, coreFile(audiowareCore, "audioware"), coreFile(tweakXLCore, "tweakxl"),
		coreFile(archiveXLCore, "archivexl"), coreFile(inputLoaderCore, "inputloader"), coreFile(modSettingsCore, "modsettings"),
		coreCyberCAT, coreFile(ammCore, "amm"), coreCyberScript,
		asiInstaller, multiType, red4extInstaller, redmodInstaller, ammInstaller, cetInstaller, redscriptInstaller,
		audiowareInstaller, tweakXLInstaller, iniInstaller, jsonInstaller, xmlInstaller, presetInstaller, archiveInstaller,
	}
	for _, s := range steps {
		o, ok, err := s(c)
		if err != nil {
			return outcome{}, err
		}
		if ok {
			return o, nil
		}
	}
	return outcome{kind: "fallback", moves: same(c.t.files)}, nil
}

// unresolvable is an installer that recognised the archive's kind but found no layout it knows: the archive is laid
// out as packed, with a note, where Vortex would ask whether to proceed.
func unresolvable(c *cpCtx, kind string) (outcome, bool, error) {
	c.note(NoteUnrecognised, "this looks like a "+kind+" mod, but its layout is not one Mortar knows, so every file is placed as packed", c.t.files...)
	return outcome{kind: "unresolvable", moves: same(c.t.files)}, true, nil
}

func (c *cpCtx) extraFiles() []move {
	var out []move
	for _, f := range c.t.files {
		first, _, _ := strings.Cut(f, "/")
		if slices.Contains(knownToplevelDirs, lower(first)) && strings.Contains(f, "/") {
			continue
		}
		if slices.Contains(extraFileExts, lower(path.Ext(f))) {
			out = append(out, move{src: f, dst: path.Join(ExtraFilesBase, c.name, f)})
		}
	}
	return out
}
