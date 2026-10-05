package contentpatcher

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
func enabledRecolours(mods []framework.Mod, self mod.ID) map[int]framework.Mod {
	out := map[int]framework.Mod{}
	for _, m := range mods {
		if !m.Enabled || mod.Equal(m.ModID(), self) {
			continue
		}
		id := squash(m.ModID().Local())
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
func recolourSettings(packMod framework.Mod, pack cachedPack, config map[string]string, enabled map[int]framework.Mod, covered map[string]bool) []framework.SettingHint {
	var out []framework.SettingHint
	for _, schema := range pack.schema {
		if schema.allowMultiple || len(schema.allowValues) < 2 || covered[strings.ToLower(schema.key)] || mapsByHasMod(pack, schema.key) {
			continue
		}
		valueFamily := map[string]int{}
		for _, v := range schema.allowValues {
			if f := familyOfValue(v); f >= 0 {
				valueFamily[v] = f
			}
		}
		if len(valueFamily) == 0 {
			continue
		}
		current, set := config[strings.ToLower(schema.key)]
		if !set {
			current = schema.defaultValue
		}
		current = strings.TrimSpace(current)
		if current == "" || slices.Contains(autoChoices, strings.ToLower(current)) {
			continue
		}
		currentFamily := familyOfValue(current)
		if _, ok := enabled[currentFamily]; currentFamily >= 0 && ok {
			continue
		}
		var suggested, forNames []string
		var forIDs []mod.ID
		for _, v := range schema.allowValues {
			f, ok := valueFamily[v]
			if !ok {
				continue
			}
			if im, on := enabled[f]; on {
				addSettingValue(&suggested, v)
				if !slices.Contains(forIDs, im.ModID()) {
					forIDs = append(forIDs, im.ModID())
					forNames = append(forNames, cmp.Or(im.Name, im.ModID().Local()))
				}
			}
		}
		currentFor := ""
		if currentFamily >= 0 {
			currentFor = recolourFamilies[currentFamily].name
			if len(suggested) == 0 {
				for _, v := range schema.allowValues {
					if s := squash(v); s == "vanilla" || s == "default" {
						suggested = []string{v}
						break
					}
				}
			}
		}
		if len(suggested) == 0 {
			continue
		}
		out = append(out, framework.SettingHint{
			Key: packMod.Key, ID: packMod.ModID(), Name: packMod.Name, Field: schema.key, Current: current,
			Suggested: suggested, Description: schema.description, Variant: true, CurrentFor: currentFor,
			For: forIDs, ForNames: forNames,
		})
	}
	return out
}

// mapsByHasMod reports whether the pack ties the field to installed mods itself, so it already follows the profile.
func mapsByHasMod(pack cachedPack, field string) bool {
	token := "{{" + strings.ToLower(field) + "}}"
	return slices.ContainsFunc(pack.patches, func(p cpPatch) bool {
		if len(p.when.anyOf) == 0 {
			return false
		}
		return strings.Contains(strings.ToLower(p.tokenValue), token) ||
			slices.ContainsFunc(p.when.config, func(c cpConfig) bool { return strings.EqualFold(c.field, field) })
	})
}

// Name words that mark a mod as made for one recolour. Broader aliases ("vibrant", "eemie") also name unrelated
// mods, so only these are trusted in a mod's name.
var recolourNameWords = map[string]int{"earthy": 0, "starblue": 1, "vpr": 2, "wittily": 3}

// recolourAddons lists enabled mods made for a recolour the profile does not have enabled, such as "Earthy Icons
// for Worldmaps Everywhere" without Earthy Recolour; their manifests rarely declare the recolour.
func recolourAddons(mods []framework.Mod) []framework.Cleanup {
	enabled := enabledRecolours(mods, "")
	var out []framework.Cleanup
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		id := squash(m.ModID().Local())
		for _, word := range strings.FieldsFunc(strings.ToLower(m.Name), func(r rune) bool { return r < 'a' || r > 'z' }) {
			f, ok := recolourNameWords[word]
			if !ok {
				continue
			}
			if _, on := enabled[f]; on || slices.ContainsFunc(recolourFamilies[f].idParts, func(p string) bool { return strings.Contains(id, p) }) {
				break
			}
			out = append(out, framework.Cleanup{Key: m.Key, ID: m.ModID(), Name: m.Name, Reason: "Made for " + recolourFamilies[f].name + ", which is not enabled"})
			break
		}
	}
	return out
}
