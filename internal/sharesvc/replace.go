package sharesvc

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// ReplacePlan is what Replace would remove, and which local-only mods it keeps.
type ReplacePlan struct {
	RemoveKeys []string `json:"removeKeys"`
	Remove     []string `json:"remove"`
	KeepLocal  []string `json:"keepLocal"`
}

func sourceToken(kind string, modID, fileID int, repo, tag, asset string) string {
	if repo != "" {
		return strings.ToLower(fmt.Sprintf("g:%s@%s/%s", repo, tag, asset))
	}
	if modID != 0 {
		return fmt.Sprintf("n:%d:%d", modID, fileID)
	}
	if kind == profile.KindLocal {
		return ""
	}
	return ""
}

func entryToken(e profile.Entry) string {
	return sourceToken(e.Source.Kind, e.Source.ModID, e.Source.FileID, e.Source.Repo, e.Source.Tag, e.Source.Asset)
}

func modToken(m Mod) string {
	return sourceToken(m.Site, m.ModID, m.FileID, m.Repo, m.Tag, m.Asset)
}

func shareIndex(mods []Mod) (files, ids map[string]struct{}) {
	files, ids = map[string]struct{}{}, map[string]struct{}{}
	for _, m := range mods {
		if tok := modToken(m); tok != "" {
			files[tok] = struct{}{}
		}
		for _, id := range m.UniqueIDs {
			ids[strings.ToLower(id)] = struct{}{}
		}
	}
	return files, ids
}

func entryLabel(e profile.Entry) string {
	if len(e.Mods) > 0 && e.Mods[0].Name != "" {
		return e.Mods[0].Name
	}
	if len(e.Mods) > 0 {
		return e.Mods[0].UniqueID
	}
	return e.Key
}

func idsOf(e profile.Entry) []string {
	out := make([]string, 0, len(e.Mods))
	for _, m := range e.Mods {
		out = append(out, strings.ToLower(m.UniqueID))
	}
	return out
}

// PlanReplace lists entries Replace would drop so the profile matches the share, keeping local-only mods.
func PlanReplace(p profile.Profile, mods []Mod) ReplacePlan {
	files, ids := shareIndex(mods)
	plan := ReplacePlan{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		tok := entryToken(e)
		if _, ok := files[tok]; ok && tok != "" {
			continue
		}
		inShare := false
		for _, id := range idsOf(e) {
			if _, ok := ids[id]; ok {
				inShare = true
				break
			}
		}
		if e.Source.Kind == profile.KindLocal && !inShare {
			plan.KeepLocal = append(plan.KeepLocal, entryLabel(e))
			continue
		}
		if inShare || tok != "" {
			plan.RemoveKeys = append(plan.RemoveKeys, e.Key)
			plan.Remove = append(plan.Remove, entryLabel(e))
		}
	}
	return plan
}
