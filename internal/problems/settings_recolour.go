package problems

import (
	"cmp"
	"slices"
	"strings"
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
func enabledRecolours(mods []Installed, self string) map[int]Installed {
	out := map[int]Installed{}
	for _, m := range mods {
		if !m.Enabled || sameID(m.UniqueID, self) {
			continue
		}
		id := squash(m.UniqueID)
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
func recolourSettings(packMod Installed, pack cachedPack, config map[string]string, enabled map[int]Installed, covered map[string]bool) []SettingHint {
	var out []SettingHint
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
		var suggested, forIDs, forNames []string
		for _, v := range schema.allowValues {
			f, ok := valueFamily[v]
			if !ok {
				continue
			}
			if mod, on := enabled[f]; on {
				addSettingValue(&suggested, v)
				if !slices.Contains(forIDs, mod.UniqueID) {
					forIDs = append(forIDs, mod.UniqueID)
					forNames = append(forNames, cmp.Or(mod.Name, mod.UniqueID))
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
		out = append(out, SettingHint{
			Key: packMod.Key, UniqueID: packMod.UniqueID, Name: packMod.Name, Field: schema.key, Current: current,
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
