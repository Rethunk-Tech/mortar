package launch

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// SMAPIProblemKind is a recognised class of SMAPI log error.
type SMAPIProblemKind string

const (
	SMAPIProblemMissingDependency SMAPIProblemKind = "missingDependency"
	SMAPIProblemTooOld            SMAPIProblemKind = "tooOld"
	SMAPIProblemContentPack       SMAPIProblemKind = "contentPack"
	SMAPIProblemDuplicate         SMAPIProblemKind = "duplicate"
	SMAPIProblemFailedToLoad      SMAPIProblemKind = "failedToLoad"
	// SMAPIProblemDependencyFailed is a mod skipped because a mod it needs is installed but was itself skipped.
	SMAPIProblemDependencyFailed SMAPIProblemKind = "dependencyFailed"
	SMAPIProblemDependencyTooOld SMAPIProblemKind = "dependencyTooOld"
	SMAPIProblemSMAPITooOld      SMAPIProblemKind = "smapiTooOld"
	SMAPIProblemGameTooOld       SMAPIProblemKind = "gameTooOld"
	SMAPIProblemObsolete         SMAPIProblemKind = "obsolete"
	SMAPIProblemHarmonyPatch     SMAPIProblemKind = "harmonyPatch"
	SMAPIProblemMalicious        SMAPIProblemKind = "malicious"
)

// SMAPIFixKind is the one-click action offered for a finding.
type SMAPIFixKind string

const (
	SMAPIFixInstallDependency SMAPIFixKind = "installDependency"
	SMAPIFixUpdate            SMAPIFixKind = "update"
	SMAPIFixDisable           SMAPIFixKind = "disable"
	SMAPIFixRemoveDuplicate   SMAPIFixKind = "removeDuplicate"
	SMAPIFixUpdateLoader      SMAPIFixKind = "updateLoader"
	SMAPIFixRemove            SMAPIFixKind = "remove"
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
		p, ok := matchSMAPIProblem(e)
		if !ok {
			continue
		}
		// One row per mod and kind: a content pack can fail dozens of patches, and the fix is the same for all.
		key := string(p.Kind) + "\x00" + p.ModID + "\x00" + p.Dependency
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

// ResolveSMAPIProblemMods fills the id and name from installed mods when the log only named the display name.
func ResolveSMAPIProblemMods(problems []SMAPIProblem, mods []ModRef) []SMAPIProblem {
	if len(problems) == 0 {
		return problems
	}
	out := append([]SMAPIProblem(nil), problems...)
	for i, p := range out {
		if m, ok := matchModRef(p.ModID, p.ModName, mods); ok {
			out[i].ModID = m.ID.Local()
			if p.ModName == "" || strings.EqualFold(p.ModName, p.ModID) {
				out[i].ModName = m.Name
			}
		}
	}
	return out
}

func matchModRef(id, name string, mods []ModRef) (ModRef, bool) {
	for _, m := range mods {
		if id != "" && mod.Equal(m.ID, mod.SMAPI(id)) {
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

func matchSMAPIProblem(e Entry) (SMAPIProblem, bool) {
	trimmed := strings.TrimSpace(e.Message)
	line := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	if line == "" {
		return SMAPIProblem{}, false
	}
	var skipped SMAPIProblem
	if e.Mod == "SMAPI" && strings.HasPrefix(trimmed, "- ") {
		var known bool
		if skipped, known = matchSkipped(line); known {
			return skipped, true
		}
	}
	if p, ok := matchModMessage(e.Mod, line); ok {
		return p, true
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
	return skipped, skipped.Kind != ""
}

// matchSkipped reads one entry of SMAPI's "Skipped mods" list, "<name> <version> because <reason>" (LogManager), where
// the reason is one of ModResolver's failure phrases. known is false for a reason it does not name; p is then the
// failed-to-load fallback, since every skipped mod failed to load, for the older wordings to try first.
func matchSkipped(line string) (p SMAPIProblem, known bool) {
	name, reason, ok := strings.Cut(line, " because ")
	if !ok || name == "" {
		return SMAPIProblem{}, false
	}
	p = SMAPIProblem{Kind: SMAPIProblemFailedToLoad, ModID: name, ModName: name, Detail: line, Fix: SMAPIFixDisable}
	set := func(kind SMAPIProblemKind, fix SMAPIFixKind, dep string) (SMAPIProblem, bool) {
		p.Kind, p.Fix, p.Dependency = kind, fix, dep
		return p, true
	}
	switch {
	case strings.HasPrefix(reason, "it requires mods which aren't installed ("):
		return set(SMAPIProblemMissingDependency, SMAPIFixInstallDependency, firstListed(reason, "("))
	case strings.HasPrefix(reason, "it needs the '"):
		dep, _, _ := cutBetween(reason, "it needs the '", "' mod")
		return set(SMAPIProblemDependencyFailed, SMAPIFixUpdate, dep)
	case strings.HasPrefix(reason, "it needs newer versions of some mods: "):
		dep, _, _ := strings.Cut(firstListed(reason, ": "), " (needs")
		return set(SMAPIProblemDependencyTooOld, SMAPIFixUpdate, dep)
	case strings.HasPrefix(reason, "it needs SMAPI "):
		return set(SMAPIProblemSMAPITooOld, SMAPIFixUpdateLoader, "")
	case strings.HasPrefix(reason, "it needs Stardew Valley "):
		return set(SMAPIProblemGameTooOld, SMAPIFixDisable, "")
	case strings.HasPrefix(reason, "it's obsolete"):
		return set(SMAPIProblemObsolete, SMAPIFixDisable, "")
	case strings.Contains(reason, ". Please check for a "):
		return set(SMAPIProblemTooOld, SMAPIFixUpdate, "")
	case strings.HasPrefix(reason, "you have multiple copies"),
		strings.HasPrefix(reason, "its DLL couldn't be loaded") && strings.Contains(reason, "because it was already loaded"):
		return set(SMAPIProblemDuplicate, SMAPIFixRemoveDuplicate, "")
	case strings.HasPrefix(reason, "its "), strings.HasPrefix(reason, "it doesn't have"):
		return p, true
	}
	return p, false
}

// firstListed is the first entry of the comma-separated list after open, without the ": <url>" SMAPI adds to a name it
// knows a page for.
func firstListed(s, open string) string {
	_, list, _ := strings.Cut(s, open)
	list = strings.TrimSuffix(strings.TrimSuffix(list, "."), ")")
	first, _, _ := strings.Cut(list, ", ")
	name, _, _ := strings.Cut(first, ": ")
	return strings.TrimSpace(name)
}

var (
	// Content Patcher names a patch "<pack> > <patch path>".
	cpPatchRe   = regexp.MustCompile(`^(?:Can't apply (?:\w+ )?patch "|Ignored )(.+?) > `)
	fsPackRe    = regexp.MustCompile(`^Unable to add \w+ for .+ from (.+?): |^Content pack (.+?) is missing `)
	maliciousRe = regexp.MustCompile(`^The '(.+)' mod has been flagged as a malicious mod\.`)
	harmonyRe   = regexp.MustCompile(`(?i)HarmonyException|Patching exception|failed to apply (?:harmony )?patch`)
)

// matchModMessage reads the errors a mod logs under its own name, and the per-mod lines SMAPI writes outside the
// skipped list.
func matchModMessage(mod, line string) (SMAPIProblem, bool) {
	named := func(kind SMAPIProblemKind, fix SMAPIFixKind, name string) (SMAPIProblem, bool) {
		return SMAPIProblem{Kind: kind, ModID: name, ModName: name, Detail: line, Fix: fix}, true
	}
	ownName := mod != "" && mod != "SMAPI" && mod != "game"
	switch {
	case mod == "SMAPI" && maliciousRe.MatchString(line):
		return named(SMAPIProblemMalicious, SMAPIFixRemove, maliciousRe.FindStringSubmatch(line)[1])
	case ownName && strings.HasPrefix(line, "Mod crashed on entry"):
		return named(SMAPIProblemFailedToLoad, SMAPIFixDisable, mod)
	case ownName && harmonyRe.MatchString(line):
		return named(SMAPIProblemHarmonyPatch, SMAPIFixUpdate, mod)
	case mod == "Content Patcher" && cpPatchRe.MatchString(line):
		return named(SMAPIProblemContentPack, SMAPIFixDisable, cpPatchRe.FindStringSubmatch(line)[1])
	case mod == "Fashion Sense" && fsPackRe.MatchString(line):
		m := fsPackRe.FindStringSubmatch(line)
		return named(SMAPIProblemContentPack, SMAPIFixDisable, m[1]+m[2])
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

// smapiNotices are SMAPI's own section headings and advisories: they introduce lines the rules read, or say
// something Mortar shows elsewhere (mod and SMAPI updates), so none of them is a gap.
var smapiNotices = []string{
	"Skipped mods",
	"These mods could not be added to your game.",
	"Changed save serializer",
	"These mods change the save serializer.",
	"you uninstall these mods.",
	"Patched game code without Harmony fix",
	"You can update ",
	"A new version of SMAPI was detected",
	"You disabled update checks",
	"You disabled mod blacklist updates",
	"this version of SMAPI is only compatible up to Stardew Valley",
	"but the oldest supported version is",
}

var updateNoticeRe = regexp.MustCompile(`: https?://\S+ \(you have \S+\)$`)

// Classified reports whether a Mortar rule accounts for a WARN, ERROR or ALERT entry: a run problem, a crash, an
// error a mod logged under its own name (which the run's error rows count), or a SMAPI notice.
func Classified(e Entry) bool {
	if _, ok := matchSMAPIProblem(e); ok || IsCrash(e) {
		return true
	}
	if e.Mod != "SMAPI" && e.Mod != "game" && e.Level != Warn {
		return true
	}
	if e.Mod != "SMAPI" {
		return false
	}
	msg := strings.TrimSpace(e.Message)
	if strings.Trim(msg, "-") == "" || updateNoticeRe.MatchString(msg) {
		return true
	}
	// A bare "- <mod>" lists a mod under the notice heading above it.
	if item, ok := strings.CutPrefix(msg, "- "); ok && !strings.Contains(item, " because ") {
		return true
	}
	for _, n := range smapiNotices {
		if strings.HasPrefix(msg, n) || strings.Contains(msg, n) && strings.HasPrefix(msg, "Oops!") {
			return true
		}
	}
	return false
}

// Unclassified counts the WARN, ERROR and ALERT messages in a SMAPI log that no rule classifies, so a run record shows
// how much of its log Mortar could not explain.
func Unclassified(log string) int {
	n := 0
	for _, e := range ParseLog(log) {
		if !e.Cont && (e.Level == Warn || e.Level == Error || e.Level == Alert) && !Classified(e) {
			n++
		}
	}
	return n
}
