package contentpatcher

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
)

// recolourFamily is a map recolour that packs offer as a config choice. Packs spell the choice their own way
// ("Earthy", "DNEarthy", "VPR"), so aliases are compared with case and punctuation stripped; idPart picks the
// recolour itself, not its add-ons ("Earthy Map for SVE" does not recolour the base game).
type recolourFamily struct {
	name    string
	idParts []string
	aliases []string
}

var recolourFamilies = []recolourFamily{
	{name: "Earthy Recolour", idParts: []string{"earthyrecolo"}, aliases: []string{"earthy", "dnearthy", "dnearthyrecolour", "earthyrecolour", "earthyrecolor"}},
	{name: "Starblue Valley", idParts: []string{"starblue"}, aliases: []string{"starblue", "starbluevalley"}},
	{name: "Vibrant Pastoral Recolor", idParts: []string{"vibrantpastoral"}, aliases: []string{"vpr", "vibrant", "vibrantpastoral", "vibrantpastoralrecolor", "vibrantpastoralrecolour"}},
	{name: "Wittily Named Recolor", idParts: []string{"wittily"}, aliases: []string{"wittily", "wittilynamed", "wittilynamedrecolor"}},
	{name: "Eemie's Map Recolour", idParts: []string{"eemie"}, aliases: []string{"eemie", "eemies", "eemiesmaprecolour"}},
}

// Values that leave the choice to the pack, which then detects the installed recolour itself.
var autoChoices = []string{"auto", "automatic", "detect", "off", "none"}

func squash(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func familyOfValue(value string) int {
	v := squash(value)
	return slices.IndexFunc(recolourFamilies, func(f recolourFamily) bool { return slices.Contains(f.aliases, v) })
}

// enabledRecolours finds, per family, the enabled mod that is that recolour.
func enabledRecolours(mods []cp.Mod, self cp.ModID) map[int]cp.Mod {
	out := map[int]cp.Mod{}
	for _, m := range mods {
		if !m.Enabled || cp.Equal(m.UniqueID, self) {
			continue
		}
		id := squash(m.UniqueID.Local())
		for i, f := range recolourFamilies {
			if _, seen := out[i]; !seen && slices.ContainsFunc(f.idParts, func(p string) bool {
				return strings.Contains(id, p) && (p != "eemie" || strings.Contains(id, "recolo"))
			}) {
				out[i] = m
			}
		}
	}
	return out
}

// recolourSettings suggests a pack's recolour choice from the recolours the profile has enabled, for packs whose
// choices name recolours without a HasMod mapping the pack could detect on its own. It also catches a choice made
// for a recolour the profile does not have enabled.
func recolourSettings(packMod cp.Mod, pack cp.Pack, config map[string]string, enabled map[int]cp.Mod, covered map[string]bool) []cp.Hint {
	var out []cp.Hint
	for _, schema := range pack.Schema {
		if schema.AllowMultiple || len(schema.AllowValues) < 2 || covered[strings.ToLower(schema.Key)] || mapsByHasMod(pack, schema.Key) {
			continue
		}
		if hint, ok := recolourHint(packMod, schema, config, enabled); ok {
			out = append(out, hint)
		}
	}
	return out
}

func recolourHint(packMod cp.Mod, schema cp.Schema, config map[string]string, enabled map[int]cp.Mod) (cp.Hint, bool) {
	valueFamily := map[string]int{}
	for _, v := range schema.AllowValues {
		if f := familyOfValue(v); f >= 0 {
			valueFamily[v] = f
		}
	}
	if len(valueFamily) == 0 {
		return cp.Hint{}, false
	}
	current, set := config[strings.ToLower(schema.Key)]
	if !set {
		current = schema.DefaultValue
	}
	current = strings.TrimSpace(current)
	if current == "" || slices.Contains(autoChoices, strings.ToLower(current)) {
		return cp.Hint{}, false
	}
	currentFamily := familyOfValue(current)
	if _, ok := enabled[currentFamily]; currentFamily >= 0 && ok {
		return cp.Hint{}, false
	}
	var suggested, forNames []string
	var forIDs []cp.ModID
	for _, v := range schema.AllowValues {
		f, ok := valueFamily[v]
		if !ok {
			continue
		}
		if im, on := enabled[f]; on {
			cp.AddSettingValue(&suggested, v)
			if !slices.Contains(forIDs, im.UniqueID) {
				forIDs = append(forIDs, im.UniqueID)
				forNames = append(forNames, cmp.Or(im.Name, im.UniqueID.Local()))
			}
		}
	}
	currentFor := ""
	if currentFamily >= 0 {
		currentFor = recolourFamilies[currentFamily].name
		if len(suggested) == 0 {
			suggested = vanillaChoice(schema.AllowValues)
		}
	}
	if len(suggested) == 0 {
		return cp.Hint{}, false
	}
	return cp.Hint{
		Key: packMod.Key, ID: packMod.UniqueID, Name: packMod.Name, Field: schema.Key, Current: current,
		Suggested: suggested, Description: schema.Description, Variant: true, CurrentFor: currentFor,
		For: forIDs, ForNames: forNames,
	}, true
}

// vanillaChoice is the first allowed value that means the game's own look, as a one-value suggestion.
func vanillaChoice(allowed []string) []string {
	for _, v := range allowed {
		if s := squash(v); s == "vanilla" || s == "default" {
			return []string{v}
		}
	}
	return nil
}

// mapsByHasMod reports whether the pack ties the field to installed mods itself, so it already follows the profile.
func mapsByHasMod(pack cp.Pack, field string) bool {
	token := "{{" + strings.ToLower(field) + "}}"
	return slices.ContainsFunc(pack.Patches, func(p cp.Patch) bool {
		if len(p.When.AnyOf) == 0 {
			return false
		}
		return strings.Contains(strings.ToLower(p.TokenValue), token) ||
			slices.ContainsFunc(p.When.Config, func(c cp.Config) bool { return strings.EqualFold(c.Field, field) })
	})
}

// Name words that mark a mod as made for one recolour. Broader aliases ("vibrant", "eemie") also name unrelated
// mods, so only these are trusted in a mod's name.
var recolourNameWords = map[string]int{"earthy": 0, "starblue": 1, "vpr": 2, "wittily": 3}

// recolourAddons lists enabled mods made for a recolour the profile does not have enabled, such as "Earthy Icons
// for Worldmaps Everywhere" without Earthy Recolour; their manifests rarely declare the recolour.
func recolourAddons(mods []cp.Mod) []framework.Cleanup {
	enabled := enabledRecolours(mods, "")
	var out []framework.Cleanup
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		id := squash(m.UniqueID.Local())
		for _, word := range strings.FieldsFunc(strings.ToLower(m.Name), func(r rune) bool { return r < 'a' || r > 'z' }) {
			f, ok := recolourNameWords[word]
			if !ok {
				continue
			}
			if _, on := enabled[f]; on || slices.ContainsFunc(recolourFamilies[f].idParts, func(p string) bool { return strings.Contains(id, p) }) {
				break
			}
			out = append(out, framework.Cleanup{
				Key: m.Key, ID: frameworkID(m.UniqueID), Name: m.Name, Reason: "Made for " + recolourFamilies[f].name + ", which is not enabled",
				Code: "recolourMissing", Params: map[string]string{"name": recolourFamilies[f].name},
			})
			break
		}
	}
	return out
}
