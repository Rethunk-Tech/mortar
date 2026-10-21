package pack

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
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
// the loader files Mortar writes itself (doorstop_config.ini), so only a file below one of roots, the game's loaders'
// import folders, is kept, written under the root's own spelling; anything else is dropped without a word, as nothing
// the player chose is lost. A file the pack lists twice keeps its config. programs lists, sorted, the executables left
// out, which the player should hear about.
func ProfileFiles(d Draft, roots []string) (out map[string][]byte, programs []string) {
	out = map[string][]byte{}
	keep := func(p string, data []byte) {
		w, ok := winPath(p)
		if ok {
			first, rest, cut := strings.Cut(w, "/")
			i := slices.IndexFunc(roots, func(r string) bool { return strings.EqualFold(r, first) })
			if !cut || i < 0 {
				return
			}
			if slices.ContainsFunc(blockedExtensions, func(e string) bool { return strings.HasSuffix(strings.ToLower(w), e) }) {
				programs = append(programs, p)
				return
			}
			out[roots[i]+"/"+rest] = data
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

// ImportRoots are the folders the game's loaders let an imported pack write loose files into.
func ImportRoots(gameID string) []string {
	info, ok := components.Game(gameID)
	if !ok {
		return nil
	}
	var out []string
	for _, l := range loader.For(info) {
		if r, ok := l.(loader.ImportRoots); ok {
			out = append(out, r.ImportRoots()...)
		}
	}
	return out
}
