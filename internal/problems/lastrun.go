package problems

import (
	"strings"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

// RunError is an enabled mod that logged errors in the profile's most recent stored run.
type RunError struct {
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	First    string `json:"first"`
	Severe   bool   `json:"severe"`
	RunID    string `json:"runId"`
}

// RunReader loads the newest recorded run's SMAPI log summary for a profile.
type RunReader interface {
	LastRunSummary(gameID, profileID string) (runID string, summary launch.Summary, err error)
}

func modRefs(mods []Installed) []launch.ModRef {
	out := make([]launch.ModRef, 0, len(mods))
	for _, m := range mods {
		out = append(out, launch.ModRef{Name: m.Name, UniqueID: m.UniqueID})
	}
	return out
}

func installedByRef(mods []Installed, ref launch.ModRef) (Installed, bool) {
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		if m.Name == ref.Name && strings.EqualFold(m.UniqueID, ref.UniqueID) {
			return m, true
		}
	}
	return Installed{}, false
}

// RunErrorsFromSummary maps a run summary to profile mods that logged errors and are still enabled.
func RunErrorsFromSummary(runID string, summary launch.Summary, mods []Installed) []RunError {
	if runID == "" || len(summary.Mods) == 0 {
		return []RunError{}
	}
	refs := modRefs(mods)
	severe := summary.Crashed
	out := []RunError{}
	for _, me := range summary.Mods {
		ref, ok := launch.MatchModColumn(me.Mod, refs)
		if !ok {
			continue
		}
		inst, ok := installedByRef(mods, ref)
		if !ok {
			continue
		}
		out = append(out, RunError{
			Key: inst.Key, UniqueID: inst.UniqueID, Name: inst.Name,
			Count: me.Count, First: me.First, Severe: severe, RunID: runID,
		})
	}
	return out
}
