package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// RunError is an enabled mod that logged errors in the profile's most recent stored run.
type RunError struct {
	Key     string `json:"key"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Count   int    `json:"count"`
	First   string `json:"first"`
	Severe  bool   `json:"severe"`
	RunID   string `json:"runId"`
	Updated bool   `json:"updated,omitempty"`
}

// RunReader loads the newest recorded run's id and SMAPI log summary for a profile.
type RunReader interface {
	LastRunID(gameID, profileID string) (string, error)
	LastRunSummary(gameID, profileID string) (runID string, summary launch.Summary, err error)
}

func modRefs(mods []framework.Mod) []launch.ModRef {
	out := make([]launch.ModRef, 0, len(mods))
	for _, m := range mods {
		out = append(out, launch.ModRef{Name: m.Name, ID: m.ModID()})
	}
	return out
}

func installedByRef(mods []framework.Mod, ref launch.ModRef) (framework.Mod, bool) {
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		if ref.Key != "" && m.Key == ref.Key && (ref.ID == "" || mod.Equal(m.ModID(), ref.ID)) &&
			(ref.Name == "" || m.Name == ref.Name) {
			return m, true
		}
		if (ref.ID != "" && mod.Equal(m.ModID(), ref.ID)) ||
			(ref.ID == "" && ref.Name != "" && m.Name == ref.Name) {
			return m, true
		}
	}
	return framework.Mod{}, false
}

func changedSinceRun(now framework.Mod, then launch.ModRef) bool {
	if then.Key == "" {
		return false
	}
	return now.Key != then.Key ||
		then.Version != "" && now.Version != then.Version ||
		then.SourceVersion != "" && now.SourceVersion != then.SourceVersion
}

// RunErrorsFromSummary maps a run summary to profile mods that logged errors and are still enabled. A column naming no
// mod is looked up, lower-cased, in owners: a BepInEx line names its plugin, and owners maps plugin names and GUIDs to
// the enabled package shipping them (see pluginOwners). owners is called only for such a column.
func RunErrorsFromSummary(runID string, summary launch.Summary, mods []framework.Mod, owners func() map[string]framework.Mod) []RunError {
	if runID == "" || len(summary.Mods) == 0 {
		return []RunError{}
	}
	refs := summary.ModRefs
	snapshot := len(refs) > 0
	if !snapshot {
		refs = modRefs(mods)
	}
	severe := summary.Crashed
	out := []RunError{}
	for _, me := range summary.Mods {
		var inst framework.Mod
		ref, ok := launch.MatchModColumn(me.Mod, refs)
		if ok {
			inst, ok = installedByRef(mods, ref)
		} else if owners != nil {
			inst, ok = owners()[strings.ToLower(strings.TrimSpace(me.Mod))]
			if i := slices.IndexFunc(refs, func(r launch.ModRef) bool { return r.Key == inst.Key }); ok && i >= 0 {
				ref = refs[i]
			}
		}
		if !ok {
			continue
		}
		out = append(out, RunError{
			Key: inst.Key, ID: inst.ModID(), Name: inst.Name,
			Count: me.Count, First: me.First, Severe: severe, RunID: runID,
			Updated: snapshot && changedSinceRun(inst, ref),
		})
	}
	return out
}

// ranLast reports whether profileID has a recorded run newer than every other profile's. Run ids start with their
// start time in a fixed-width form, so they order as strings.
func ranLast(runs RunReader, gameID, profileID string, others []string) bool {
	mine, err := runs.LastRunID(gameID, profileID)
	if err != nil || mine == "" {
		return false
	}
	for _, o := range others {
		if theirs, err := runs.LastRunID(gameID, o); err == nil && theirs > mine {
			return false
		}
	}
	return true
}
