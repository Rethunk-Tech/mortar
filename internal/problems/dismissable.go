package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Dismissable is one counted problem a user can dismiss, in the order the CLI numbers them.
type Dismissable struct {
	Kind         string // "listed", "abandoned", "conflict", "setting" or "loadFailure"
	ID           mod.ID
	Field        string
	ConflictKind string
	Target       string
	// Failure names a loadFailure row: the log's finding, which Service.DismissLoadFailure hides for the newest run.
	Failure LoadFailure
}

// DismissableRows lists the result's dismissable problems: listed requirements, abandoned, deprecated and
// compatibility-listed mods, counted conflicts, compatibility settings and the rows read from a run's log.
func DismissableRows(r Result) []Dismissable {
	var out []Dismissable
	for _, x := range r.Missing {
		if x.Listed {
			out = append(out, Dismissable{Kind: "listed", ID: x.ID})
		}
	}
	for _, x := range r.Broken {
		if DismissibleBroken(x) {
			out = append(out, Dismissable{Kind: "abandoned", ID: x.ID})
		}
	}
	for _, x := range r.Deprecated {
		out = append(out, Dismissable{Kind: "abandoned", ID: x.ID})
	}
	for _, x := range r.Compat {
		out = append(out, Dismissable{Kind: "abandoned", ID: x.ID})
	}
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic && x.Kind != "" {
			out = append(out, Dismissable{Kind: "conflict", ConflictKind: x.Kind, Target: x.Target})
		}
	}
	for _, x := range r.Settings {
		out = append(out, Dismissable{Kind: "setting", ID: x.ID, Field: x.Field})
	}
	for _, x := range r.LoadFailures {
		if DismissibleLoadFailure(x) {
			out = append(out, Dismissable{Kind: "loadFailure", Failure: x})
		}
	}
	return out
}

// DismissibleLoadFailure is a row from a run's log; the game's version and settings rows come from the install.
func DismissibleLoadFailure(f LoadFailure) bool {
	return f.Kind != KindGameVersion && f.Kind != KindGameSetting
}

// DismissibleBroken is advice rather than a fault to fix: an abandoned, obsolete or deprecated mod, or one its source
// flags.
func DismissibleBroken(b Broken) bool {
	return b.Status == "abandoned" || b.Status == "obsolete" || b.Status == "deprecated" || b.Source != ""
}

func dismissBucket(gameID, profileID string) string {
	return "~mortar/cp/" + gameID + "/" + profileID
}

func dismissToken(kind, target string) string {
	return kind + "\t" + target
}

func settingChoiceToken(uniqueID mod.ID, field, value string) string {
	target := uniqueID.Fold() + "\t" +
		strings.ToLower(strings.TrimSpace(field)) + "\t" +
		strings.ToLower(strings.TrimSpace(value))
	return dismissToken("setting-choice", target)
}

// redundantToken names one Redundant row by its kind, its mod and the mods that make it redundant, so the row comes
// back when that set changes.
func redundantToken(kind, key string, by []string) string {
	by = slices.Clone(by)
	slices.Sort(by)
	return dismissToken("redundant", kind+"\t"+key+"\t"+strings.Join(by, ","))
}

// settingToken names one compatibility setting of a mod.
func settingToken(uniqueID mod.ID, field string) string {
	return dismissToken("setting", uniqueID.Fold()+"\t"+strings.ToLower(strings.TrimSpace(field)))
}

// hideRows splits rows into those still shown and those a dismissal hides. tokensOf lists the tokens that would hide
// a row, none for a row that cannot be dismissed; wrap puts a hidden row into its DismissedProblem field.
func hideRows[T any](rows []T, dismissed []string, tokensOf func(T) []string, wrap func(*T) DismissedProblem) ([]T, []DismissedProblem) {
	if len(dismissed) == 0 {
		return rows, nil
	}
	skip := make(map[string]bool, len(dismissed))
	for _, t := range dismissed {
		skip[t] = true
	}
	shown := []T{}
	hidden := []DismissedProblem{}
rows:
	for _, r := range rows {
		for _, token := range tokensOf(r) {
			if skip[token] {
				p := wrap(&r)
				p.Token = token
				hidden = append(hidden, p)
				continue rows
			}
		}
		shown = append(shown, r)
	}
	return shown, hidden
}

func hideDismissedRedundant(rows []framework.Redundant, tokens []string) ([]framework.Redundant, []DismissedProblem) {
	return hideRows(rows, tokens, func(r framework.Redundant) []string {
		by := make([]string, len(r.By))
		for i, b := range r.By {
			by[i] = b.Key
		}
		return []string{redundantToken(r.Kind, r.Key, by)}
	}, func(r *framework.Redundant) DismissedProblem { return DismissedProblem{Redundant: r} })
}

func hideDismissedBroken(broken []Broken, tokens []string) ([]Broken, []DismissedProblem) {
	return hideRows(broken, tokens, func(b Broken) []string {
		if !DismissibleBroken(b) {
			return nil
		}
		return []string{modAdviceToken(b.ID)}
	}, func(b *Broken) DismissedProblem { return DismissedProblem{Broken: b} })
}

// modAdviceToken is the one dismissal for advice about a mod's standing: an author-marked broken row, a deprecated
// package or a compatibility-list entry.
func modAdviceToken(id mod.ID) string {
	return dismissToken("broken", id.Fold())
}

func hideDismissedDeprecated(rows []DeprecatedPackage, tokens []string) ([]DeprecatedPackage, []DismissedProblem) {
	return hideRows(rows, tokens, func(d DeprecatedPackage) []string {
		return []string{modAdviceToken(d.ID)}
	}, func(d *DeprecatedPackage) DismissedProblem { return DismissedProblem{Deprecated: d} })
}

func hideDismissedCompat(rows []Compat, tokens []string) ([]Compat, []DismissedProblem) {
	return hideRows(rows, tokens, func(c Compat) []string {
		return []string{modAdviceToken(c.ID)}
	}, func(c *Compat) DismissedProblem { return DismissedProblem{Compat: c} })
}

func hideDismissedListed(missing []Missing, tokens []string) ([]Missing, []DismissedProblem) {
	return hideRows(missing, tokens, func(m Missing) []string {
		if !m.Listed {
			return nil
		}
		return []string{dismissToken("listed", m.ID.Fold())}
	}, func(m *Missing) DismissedProblem { return DismissedProblem{Missing: m} })
}

func hideDismissedSettings(settings []framework.SettingHint, tokens []string) ([]framework.SettingHint, []DismissedProblem) {
	return hideRows(settings, tokens, func(h framework.SettingHint) []string {
		return []string{settingToken(h.ID, h.Field), settingChoiceToken(h.ID, h.Field, h.Current)}
	}, func(h *framework.SettingHint) DismissedProblem { return DismissedProblem{Setting: h} })
}

func hideDismissed(conflicts []framework.AssetConflict, tokens []string) ([]framework.AssetConflict, []DismissedProblem) {
	return hideRows(conflicts, tokens, func(c framework.AssetConflict) []string {
		return []string{dismissToken(c.Kind, c.Target)}
	}, func(c *framework.AssetConflict) DismissedProblem { return DismissedProblem{AssetConflict: c} })
}

const loadFailureKind = "load-failure"

// loadFailureToken names one load failure of one run, so the dismissal ends when the next run records a new id.
func loadFailureToken(runID string, f LoadFailure) string {
	return dismissToken(loadFailureKind, strings.Join([]string{runID, f.Kind, f.Plugin, f.Message}, "\t"))
}

func hideDismissedLoadFailures(rows []LoadFailure, runID string, tokens []string) ([]LoadFailure, []DismissedProblem) {
	if runID == "" {
		return rows, nil
	}
	return hideRows(rows, tokens, func(f LoadFailure) []string {
		return []string{loadFailureToken(runID, f)}
	}, func(f *LoadFailure) DismissedProblem { return DismissedProblem{LoadFailure: f} })
}
