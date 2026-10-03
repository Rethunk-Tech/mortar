package launch

import (
	"strings"
)

// SMAPIProblemKind is a recognised class of SMAPI log error.
type SMAPIProblemKind string

const (
	SMAPIProblemMissingDependency SMAPIProblemKind = "missingDependency"
	SMAPIProblemTooOld            SMAPIProblemKind = "tooOld"
	SMAPIProblemContentPack       SMAPIProblemKind = "contentPack"
	SMAPIProblemDuplicate         SMAPIProblemKind = "duplicate"
	SMAPIProblemFailedToLoad      SMAPIProblemKind = "failedToLoad"
)

// SMAPIFixKind is the one-click action offered for a finding.
type SMAPIFixKind string

const (
	SMAPIFixInstallDependency SMAPIFixKind = "installDependency"
	SMAPIFixUpdate            SMAPIFixKind = "update"
	SMAPIFixDisable           SMAPIFixKind = "disable"
	SMAPIFixRemoveDuplicate   SMAPIFixKind = "removeDuplicate"
)

// SMAPI problem phrases copied from SMAPI's own log lines (ModResolver / load errors / Content Patcher).
const (
	smapiNeedsInstalled     = "because it needs "
	smapiWhichIsntInstalled = "which isn't installed"
	smapiNoLongerCompatible = "is no longer compatible"
	smapiNewerSMAPI         = "requires a newer version of SMAPI"
	smapiNewerGame          = "requires a newer version of"
	smapiMultipleCopies     = "You have multiple copies of this mod installed"
	smapiMultipleCopiesAlt  = "multiple copies of this mod installed"
	smapiCantLoadPack       = "Can't load content pack"
	smapiCantApplyPatch     = "Can't apply patch"
	smapiFailedLoadingThe   = "Failed loading the '"
)

// SMAPIProblem is one recognised issue in a run's SMAPI log.
type SMAPIProblem struct {
	Kind       SMAPIProblemKind `json:"kind"`
	ModID      string           `json:"modId"`
	ModName    string           `json:"modName"`
	Detail     string           `json:"detail"`
	Fix        SMAPIFixKind     `json:"fix"`
	Dependency string           `json:"dependency,omitempty"`
}

// ParseSMAPIProblems scans a SMAPI log for common load errors and the fix each one offers.
func ParseSMAPIProblems(log string) []SMAPIProblem {
	var out []SMAPIProblem
	seen := map[string]struct{}{}
	for _, e := range ParseLog(log) {
		p, ok := matchSMAPIProblem(e.Message)
		if !ok {
			continue
		}
		key := string(p.Kind) + "\x00" + p.ModID + "\x00" + p.Dependency + "\x00" + p.Detail
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	if out == nil {
		return []SMAPIProblem{}
	}
	return out
}

// ResolveSMAPIProblemMods fills UniqueID/name from installed mods when the log only named the display name.
func ResolveSMAPIProblemMods(problems []SMAPIProblem, mods []ModRef) []SMAPIProblem {
	if len(problems) == 0 {
		return problems
	}
	out := append([]SMAPIProblem(nil), problems...)
	for i, p := range out {
		if m, ok := matchModRef(p.ModID, p.ModName, mods); ok {
			out[i].ModID = m.UniqueID
			if p.ModName == "" || strings.EqualFold(p.ModName, p.ModID) {
				out[i].ModName = m.Name
			}
		}
	}
	return out
}

func matchModRef(id, name string, mods []ModRef) (ModRef, bool) {
	for _, m := range mods {
		if id != "" && strings.EqualFold(m.UniqueID, id) {
			return m, true
		}
		if name != "" && strings.EqualFold(m.Name, name) {
			return m, true
		}
		if name != "" && strings.HasPrefix(strings.ToLower(name), strings.ToLower(m.Name)+" ") {
			return m, true
		}
	}
	return ModRef{}, false
}

func matchSMAPIProblem(message string) (SMAPIProblem, bool) {
	line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), "-"))
	if line == "" {
		return SMAPIProblem{}, false
	}
	if p, ok := matchDuplicate(line); ok {
		return p, true
	}
	if p, ok := matchMissingDependency(line); ok {
		return p, true
	}
	if p, ok := matchTooOld(line); ok {
		return p, true
	}
	if p, ok := matchContentPack(line); ok {
		return p, true
	}
	if p, ok := matchFailedToLoad(line); ok {
		return p, true
	}
	return SMAPIProblem{}, false
}

func matchMissingDependency(line string) (SMAPIProblem, bool) {
	if !strings.Contains(line, smapiWhichIsntInstalled) {
		return SMAPIProblem{}, false
	}
	name, rest, ok := cutBetween(line, "Skipped '", smapiNeedsInstalled)
	if !ok {
		name, rest, ok = cutBetween(line, "Skipped ", smapiNeedsInstalled)
	}
	if !ok {
		name, rest, ok = strings.Cut(line, smapiNeedsInstalled)
		name = strings.TrimSpace(name)
	}
	if !ok {
		return SMAPIProblem{}, false
	}
	name = strings.Trim(strings.TrimSpace(name), "'\"")
	depPart := rest
	if i := strings.Index(depPart, " ("); i >= 0 {
		depPart = depPart[:i]
	}
	if i := strings.Index(depPart, ","); i >= 0 {
		depPart = depPart[:i]
	}
	depID := firstToken(depPart)
	if depID == "" || name == "" {
		return SMAPIProblem{}, false
	}
	return SMAPIProblem{
		Kind:       SMAPIProblemMissingDependency,
		ModID:      name,
		ModName:    name,
		Detail:     line,
		Fix:        SMAPIFixInstallDependency,
		Dependency: depID,
	}, true
}

func matchTooOld(line string) (SMAPIProblem, bool) {
	lower := strings.ToLower(line)
	compat := strings.Contains(line, smapiNoLongerCompatible) || strings.Contains(line, "it's no longer compatible")
	newerSMAPI := strings.Contains(line, smapiNewerSMAPI)
	newerGame := strings.Contains(lower, smapiNewerGame) && strings.Contains(lower, "game")
	if !compat && !newerSMAPI && !newerGame {
		return SMAPIProblem{}, false
	}
	name := leadingModName(line)
	return SMAPIProblem{
		Kind:    SMAPIProblemTooOld,
		ModID:   name,
		ModName: name,
		Detail:  line,
		Fix:     SMAPIFixUpdate,
	}, true
}

func matchContentPack(line string) (SMAPIProblem, bool) {
	cantLoad := strings.Contains(line, smapiCantLoadPack) || strings.Contains(strings.ToLower(line), "can't load content pack")
	patchErr := strings.Contains(line, smapiCantApplyPatch) || strings.Contains(strings.ToLower(line), "patch error")
	failedPack := strings.Contains(strings.ToLower(line), "failed to load") && strings.Contains(strings.ToLower(line), "content pack")
	if !cantLoad && !patchErr && !failedPack {
		return SMAPIProblem{}, false
	}
	name := contentPackName(line)
	return SMAPIProblem{
		Kind:    SMAPIProblemContentPack,
		ModID:   name,
		ModName: name,
		Detail:  line,
		Fix:     SMAPIFixDisable,
	}, true
}

func matchDuplicate(line string) (SMAPIProblem, bool) {
	if !strings.Contains(line, smapiMultipleCopies) && !strings.Contains(line, smapiMultipleCopiesAlt) {
		return SMAPIProblem{}, false
	}
	name := leadingModName(line)
	return SMAPIProblem{
		Kind:    SMAPIProblemDuplicate,
		ModID:   name,
		ModName: name,
		Detail:  line,
		Fix:     SMAPIFixRemoveDuplicate,
	}, true
}

func matchFailedToLoad(line string) (SMAPIProblem, bool) {
	lower := strings.ToLower(line)
	if strings.HasPrefix(line, smapiFailedLoadingThe) {
		name, _, ok := cutBetween(line, smapiFailedLoadingThe, "' mod")
		if !ok {
			name = leadingModName(line)
		}
		return SMAPIProblem{
			Kind:    SMAPIProblemFailedToLoad,
			ModID:   name,
			ModName: name,
			Detail:  line,
			Fix:     SMAPIFixDisable,
		}, true
	}
	if !strings.Contains(lower, "failed to load") {
		return SMAPIProblem{}, false
	}
	if strings.Contains(lower, "failed to load the game") {
		return SMAPIProblem{}, false
	}
	name := leadingModName(line)
	return SMAPIProblem{
		Kind:    SMAPIProblemFailedToLoad,
		ModID:   name,
		ModName: name,
		Detail:  line,
		Fix:     SMAPIFixDisable,
	}, true
}

func cutBetween(s, left, right string) (mid, after string, ok bool) {
	i := strings.Index(s, left)
	if i < 0 {
		return "", "", false
	}
	s = s[i+len(left):]
	j := strings.Index(s, right)
	if j < 0 {
		return "", "", false
	}
	return s[:j], s[j+len(right):], true
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "'\"")
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, "'\"")
}

func leadingModName(line string) string {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
	if strings.HasPrefix(s, "'") {
		if i := strings.Index(s[1:], "'"); i >= 0 {
			return s[1 : 1+i]
		}
	}
	if name, _, ok := cutBetween(s, "Failed loading the '", "' mod"); ok {
		return name
	}
	for _, mark := range []string{
		" failed to load",
		" is no longer compatible",
		" because it's no longer compatible",
		" because it requires a newer",
		" requires a newer",
		" because you have multiple copies",
		": You have multiple copies",
	} {
		if i := strings.Index(s, mark); i > 0 {
			return strings.Trim(s[:i], " :")
		}
	}
	return strings.TrimSpace(s)
}

func contentPackName(line string) string {
	if name, _, ok := cutBetween(line, "content pack '", "'"); ok {
		return name
	}
	if name, _, ok := cutBetween(line, `content pack "`, `"`); ok {
		return name
	}
	return leadingModName(line)
}
