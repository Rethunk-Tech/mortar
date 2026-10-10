package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Dismissable is one counted problem a user can dismiss, in the order the CLI numbers them.
type Dismissable struct {
	Kind         string // "listed", "abandoned", "conflict" or "setting"
	ID           mod.ID
	Field        string
	ConflictKind string
	Target       string
}

// DismissableRows lists the result's dismissable problems: listed requirements, abandoned mods,
// counted conflicts and compatibility settings.
func DismissableRows(r Result) []Dismissable {
	var out []Dismissable
	for _, x := range r.Missing {
		if x.Listed {
			out = append(out, Dismissable{Kind: "listed", ID: x.ID})
		}
	}
	for _, x := range r.Broken {
		if dismissibleBroken(x) {
			out = append(out, Dismissable{Kind: "abandoned", ID: x.ID})
		}
	}
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic && x.Kind != "" {
			out = append(out, Dismissable{Kind: "conflict", ConflictKind: x.Kind, Target: x.Target})
		}
	}
	for _, x := range r.Settings {
		out = append(out, Dismissable{Kind: "setting", ID: x.ID, Field: x.Field})
	}
	return out
}

// dismissibleBroken is advice rather than a fault to fix: an abandoned, obsolete or deprecated mod, or one its source
// flags.
func dismissibleBroken(b Broken) bool {
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

func hideDismissedRedundant(rows []framework.Redundant, tokens []string) ([]framework.Redundant, []DismissedProblem) {
	if len(tokens) == 0 {
		return rows, nil
	}
	var out []framework.Redundant
	dismissed := []DismissedProblem{}
	for _, r := range rows {
		by := make([]string, len(r.By))
		for i, b := range r.By {
			by[i] = b.Key
		}
		token := redundantToken(r.Kind, r.Key, by)
		if slices.Contains(tokens, token) {
			dismissed = append(dismissed, DismissedProblem{Token: token, Redundant: &r})
			continue
		}
		out = append(out, r)
	}
	return out, dismissed
}

func hideDismissedBroken(broken []Broken, tokens []string) ([]Broken, []DismissedProblem) {
	if len(tokens) == 0 {
		return broken, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Broken{}
	dismissed := []DismissedProblem{}
	for _, b := range broken {
		token := dismissToken("broken", b.ID.Fold())
		if dismissibleBroken(b) && skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Broken: &b})
			continue
		}
		out = append(out, b)
	}
	return out, dismissed
}

func hideDismissedListed(missing []Missing, tokens []string) ([]Missing, []DismissedProblem) {
	if len(tokens) == 0 {
		return missing, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Missing{}
	dismissed := []DismissedProblem{}
	for _, m := range missing {
		token := dismissToken("listed", m.ID.Fold())
		if m.Listed && skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Missing: &m})
			continue
		}
		out = append(out, m)
	}
	return out, dismissed
}

func hideDismissedSettings(settings []framework.SettingHint, tokens []string) ([]framework.SettingHint, []DismissedProblem) {
	if len(tokens) == 0 {
		return settings, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []framework.SettingHint{}
	dismissed := []DismissedProblem{}
	for _, setting := range settings {
		target := setting.ID.Fold() + "\t" + strings.ToLower(setting.Field)
		token := dismissToken("setting", target)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Setting: &setting})
			continue
		}
		token = settingChoiceToken(setting.ID, setting.Field, setting.Current)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Setting: &setting})
			continue
		}
		out = append(out, setting)
	}
	return out, dismissed
}

func hideDismissed(conflicts []framework.AssetConflict, tokens []string) ([]framework.AssetConflict, []DismissedProblem) {
	if len(tokens) == 0 {
		return conflicts, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []framework.AssetConflict{}
	dismissed := []DismissedProblem{}
	for _, c := range conflicts {
		token := dismissToken(c.Kind, c.Target)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, AssetConflict: &c})
			continue
		}
		out = append(out, c)
	}
	return out, dismissed
}
