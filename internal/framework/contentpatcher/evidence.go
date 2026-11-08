package contentpatcher

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
)

// itemConflictInfo is the place an item conflict is about, "cell (x, y)": the shared cells are 16-pixel squares
// of the target image, and the user reads pixel positions.
func itemConflictInfo(cells []string) string {
	if len(cells) == 0 {
		return ""
	}
	parts := strings.Split(cells[0], ",")
	if len(parts) == 2 {
		x, xErr := strconv.Atoi(parts[0])
		y, yErr := strconv.Atoi(parts[1])
		if xErr == nil && yErr == nil {
			return "cell (" + strconv.Itoa(x*16) + ", " + strconv.Itoa(y*16) + ")"
		}
	}
	return "cell (" + strings.ReplaceAll(cells[0], ",", ", ") + ")"
}

// whenSummary words a patch's conditions for the evidence list.
func whenSummary(w cp.When) string {
	var parts []string
	for _, group := range w.AnyOf {
		parts = append(parts, "HasMod "+strings.Join(group, ", "))
	}
	for _, id := range w.NoneOf {
		parts = append(parts, "not HasMod "+id)
	}
	for _, c := range w.Config {
		parts = append(parts, c.Field+"="+strings.Join(c.Values, "|"))
	}
	for _, d := range w.Dynamic {
		label := d.Name
		if d.Contains != "" {
			label += " contains " + d.Contains
		} else if len(d.Values) > 0 {
			label += "=" + strings.Join(d.Values, ", ")
		}
		if !d.Expected {
			label = "not " + label
		}
		parts = append(parts, label)
	}
	for _, f := range w.Flags {
		if f.Present {
			parts = append(parts, "HasFlag "+f.Name)
		} else {
			parts = append(parts, "not HasFlag "+f.Name)
		}
	}
	if w.Spouse != "" && w.Spouse != "\x00" {
		parts = append(parts, "Spouse "+w.Spouse)
	}
	for _, token := range sortedMapKeys(w.Places) {
		parts = append(parts, token+"="+strings.Join(w.Places[token], ", "))
	}
	if len(w.Assumed) > 0 {
		parts = append(parts, "also needs: "+strings.Join(compactSortedStrings(w.Assumed), ", "))
	}
	return strings.Join(parts, "; ")
}

// keyLabel words a data key as "Entry 301" or "Field 301.Price", dropping the pack scoping of token keys.
func keyLabel(key string) string {
	label := "Entry "
	if rest, ok := strings.CutPrefix(key, "field:"); ok {
		label, key = "Field ", rest
	} else {
		key = strings.TrimPrefix(key, "entry:")
	}
	return label + packScope.ReplaceAllString(key, "")
}

// packScope matches the "@<pack folder>|" packScopedKey puts before a token key.
var packScope = regexp.MustCompile(`@[^|]*\|`)

func sortedMapKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func compactSortedStrings(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
}
