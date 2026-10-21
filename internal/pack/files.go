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
// executable types are skipped. Mortar's profile folder also holds its own records (profile.json, history, saves), so
// only a file below one of roots, the game's loaders' import folders, is kept, written under the root's own spelling. A
// file the pack lists twice keeps its config. skipped lists, sorted, the paths left out.
func ProfileFiles(d Draft, roots []string) (out map[string][]byte, skipped []string) {
	out = map[string][]byte{}
	keep := func(p string, data []byte) {
		w, ok := winPath(p)
		if ok {
			first, rest, cut := strings.Cut(w, "/")
			i := slices.IndexFunc(roots, func(r string) bool { return strings.EqualFold(r, first) })
			if cut && i >= 0 && !slices.ContainsFunc(blockedExtensions, func(e string) bool { return strings.HasSuffix(strings.ToLower(w), e) }) {
				out[roots[i]+"/"+rest] = data
				return
			}
		}
		skipped = append(skipped, p)
	}
	for _, f := range d.Loose {
		keep(f.Path, f.Data)
	}
	for _, f := range d.Configs {
		keep("BepInEx/"+f.Path, f.Data)
	}
	slices.Sort(skipped)
	return out, skipped
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
