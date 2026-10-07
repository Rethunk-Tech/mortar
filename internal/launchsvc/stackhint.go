package launchsvc

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// genericSegments are namespace parts shared by many mods, which say nothing about whose code a frame is.
var genericSegments = map[string]bool{"stardew": true, "mods": true, "framework": true, "mod": true, "common": true, "stardewvalley": true}

// blamedMod is the enabled mod the last blaming exception of a log points at. A stack names code only by its
// namespace, so a mod matches when a namespace segment is its assembly's name (EntryDll), the last part of its
// UniqueID or its display name; the Harmony id of a patch is a UniqueID. Several mods matching equally blame none.
func blamedMod(blames []loader.Blame, mods []profile.Installed) (profile.Installed, loader.Blame, bool) {
	for _, b := range slices.Backward(blames) {
		best, tied, top := profile.Installed{}, false, 0
		for _, m := range mods {
			if !m.Enabled {
				continue
			}
			score := blameScore(b, m)
			switch {
			case score > top:
				best, tied, top = m, false, score
			case score == top && score > 0:
				tied = true
			}
		}
		if top > 0 && !tied {
			return best, b, true
		}
	}
	return profile.Installed{}, loader.Blame{}, false
}

func blameScore(b loader.Blame, m profile.Installed) int {
	id := strings.ToLower(m.ModID().Local())
	if b.PatchOwner != "" {
		if strings.ToLower(b.PatchOwner) == id {
			return 4
		}
		return 0
	}
	assembly := fold(strings.TrimSuffix(filepath.Base(m.EntryDll), filepath.Ext(m.EntryDll)))
	last := fold(id[strings.LastIndex(id, ".")+1:])
	name := fold(m.Name)
	score := 0
	for seg := range strings.SplitSeq(b.Namespace, ".") {
		s := fold(seg)
		if len(s) < 4 || genericSegments[s] {
			continue
		}
		switch {
		case assembly != "" && s == assembly:
			score = max(score, 3)
		case s == last:
			score = max(score, 2)
		case s == name:
			score = max(score, 1)
		}
	}
	return score
}
