package pack

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// blockedExtensions are the executable types r2modman refuses to extract from an imported profile
// (ProfileUtils.ts:44-50).
var blockedExtensions = []string{
	".dll", ".exe", ".scr", ".com", ".pif", ".bat", ".cmd", ".ps1", ".vbs", ".vbe", ".js", ".jse", ".wsf", ".wsh",
	".hta", ".msi", ".msix", ".sys", ".drv", ".cpl", ".ocx", ".lnk", ".reg", ".inf",
}

// ProfileFiles is the pack's config files and loose files as the profile holds them, by r2modman's import rule
// (ProfileUtils.ts:56-67): a config/ entry goes below BepInEx/, any other path is relative to the profile, and
// executable types are skipped. Mortar's profile folder also holds its own records (profile.json, history, saves) and
// the loader files Mortar writes itself (doorstop_config.ini), and on most loaders a mod folder is a program tree, so
// only a text file below one of roots, the game's loaders' config folders, is kept, written under the root's own
// spelling; anything else is dropped without a word, as nothing the player chose is lost. A file the pack lists twice keeps its config. programs lists, sorted, the executables left
// out, which the player should hear about. The bridge's state file is dropped too: it holds a token and port Mortar
// would otherwise trust from the pack.
func ProfileFiles(d Draft, roots []string) (out map[string][]byte, programs []string) {
	out = map[string][]byte{}
	keep := func(p string, data []byte) {
		w, ok := winPath(p)
		if ok {
			if !strings.Contains(w, "/") || strings.EqualFold(w, bridge.BepInEx.StateFile) {
				return
			}
			if blocked(w) {
				programs = append(programs, p)
				return
			}
			i := slices.IndexFunc(roots, func(r string) bool {
				return len(w) > len(r) && w[len(r)] == '/' && strings.EqualFold(w[:len(r)], r)
			})
			if i < 0 || !datadir.EditableText(w) {
				return
			}
			out[roots[i]+w[len(roots[i]):]] = data
		}
	}
	for _, f := range d.Loose {
		keep(f.Path, f.Data)
	}
	for _, f := range d.Configs {
		keep("BepInEx/"+f.Path, f.Data)
	}
	slices.Sort(programs)
	return out, programs
}

// blocked judges a file by the name it runs under: r2modman and Gale switch a mod off by adding ".old", and switching it
// back on takes that away, so "x.dll.old" is a program too.
func blocked(p string) bool {
	name := strings.ToLower(p)
	for strings.HasSuffix(name, ".old") {
		name = strings.TrimSuffix(name, ".old")
	}
	return slices.ContainsFunc(blockedExtensions, func(e string) bool { return strings.HasSuffix(name, e) })
}

// winPath is p with each segment's trailing dots and spaces removed, as Windows stores it, so "evil.dll." is judged as
// the evil.dll it becomes there. A segment with nothing else, or a name Windows cannot hold, refuses the path.
func winPath(p string) (string, bool) {
	segs := strings.Split(p, "/")
	for i, seg := range segs {
		segs[i] = strings.TrimRight(seg, ". ")
		if !winname.Valid(segs[i]) {
			return "", false
		}
	}
	return strings.Join(segs, "/"), true
}

// ImportRoots are the config folders of the game's loaders, the only places an imported pack may write loose files.
func ImportRoots(gameID string) []string {
	info, ok := components.Game(gameID)
	if !ok {
		return nil
	}
	var out []string
	for _, l := range loader.For(info) {
		if c, ok := l.(loader.WithConfig); ok {
			out = append(out, c.ConfigDirs()...)
		}
	}
	return out
}
