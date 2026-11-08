// Package cp reads the content packs written for Content Patcher: it works out what each change writes, how packs
// overlap, which pack wins a clash and which packs nothing else needs. It is Mortar's analysis without Mortar's
// profile, settings and wording: callers pass their mods in a Request and word every finding themselves.
package cp

import "strings"

const formatSMAPI = "smapi"

// ModID is "<format>:<local>", for example "smapi:Pathoschild.ContentPatcher". The zero ModID is empty.
type ModID string

// SMAPI is shorthand for a SMAPI unique id.
func SMAPI(uniqueID string) ModID { return ModID(formatSMAPI + ":" + uniqueID) }

// Format is the part before the first colon; empty when the ID has none.
func (id ModID) Format() string {
	f, _, ok := strings.Cut(string(id), ":")
	if !ok {
		return ""
	}
	return f
}

// Local is the part after the first colon, or the whole text when there is no colon.
func (id ModID) Local() string {
	_, l, ok := strings.Cut(string(id), ":")
	if !ok {
		return string(id)
	}
	return l
}

// Fold is the key an ID is mapped and compared by: SMAPI ids are case-insensitive and may carry stray spaces.
func (id ModID) Fold() string {
	if id.Format() == formatSMAPI {
		return formatSMAPI + ":" + strings.ToLower(strings.TrimSpace(id.Local()))
	}
	return string(id)
}

// Equal reports whether two ids name the same mod: formats compare exactly, locals by the format's rule.
func Equal(a, b ModID) bool {
	if a.Format() == formatSMAPI && b.Format() == formatSMAPI {
		return strings.EqualFold(strings.TrimSpace(a.Local()), strings.TrimSpace(b.Local()))
	}
	return a == b
}

// Dependency is a mod another mod needs.
type Dependency struct {
	UniqueID ModID
	Required bool
}

// Mod is one mod of the profile, with what the analysis reads from its manifest.
type Mod struct {
	Key      string
	Name     string
	UniqueID ModID
	Author   string
	Version  string
	// Folder is the mod's directory in the profile, where its content.json lives.
	Folder  string
	Enabled bool
	// Dependencies lists the manifests' Dependencies and ContentPackFor.
	Dependencies []Dependency
	// LoadAfter are the mods this one loads after when the user made it win an edit conflict.
	LoadAfter []ModID
	// ContentPackFor is the framework the mod is content for, empty for a code mod.
	ContentPackFor ModID
}

// Request is the profile to analyze and where its caches live.
type Request struct {
	Enabled, All []Mod
	// SkipImageOverlap turns off the pixel-overlap scan of image edits, trading accuracy for speed.
	SkipImageOverlap bool
	// CacheRoot is the folder the on-disk caches live under; empty turns them off.
	CacheRoot string
	// Order is the profile's load order as folded ids, or nil for none.
	Order func([]Mod) []string
	// Extend is the caller's own pass over the same parsed packs, run inside the check so its reads share the
	// check's stamps and part cache. Its findings merge into the library's, and are memoised with them.
	Extend func(run *PartsRun, all []Mod) Extension
}

// Extension is a caller's own findings from its pass over the profile's packs.
type Extension struct {
	// Hints are the caller's setting suggestions; they come before the hints that settle a clash.
	Hints []Hint
}

// WinnerKind values of a Conflict.
const (
	WinnerTop       = "top"        // the pack the checker ranks highest
	WinnerChosen    = "chosen"     // the user's win choices make one pack's edits stand
	WinnerLoadOrder = "load-order" // a dependency makes one pack load last
	WinnerNeither   = "neither"    // Content Patcher applies none of the exclusive loads
	WinnerUnclear   = "unclear"
	WinnerPerEntry  = "per-entry" // the user settled each clash with a different winner
)

// Findings is what analyzing a profile's packs finds.
type Findings struct {
	Conflicts []Conflict
	// Hints are settings that would settle a load clash.
	Hints []Hint
	// Shadowed are the packs whose every change later packs overwrite.
	Shadowed []Shadowed
	// Bundled are, by mod key, the pack by the same author that already writes everywhere each one does.
	Bundled map[string]ModRef
	// Supporting are the keys of mods that only supply assets for other mods.
	Supporting []string
	// Mentioned are the folded ids of mods the switched-on packs name in their conditions.
	Mentioned []string
	// Tilesheets are the packs that only load map and tilesheet assets, and which mods use them.
	Tilesheets []TilesheetUse
	// TilesheetsUnreadable marks a check that could not read every map, so any tilesheet might be used.
	TilesheetsUnreadable bool
}

// Conflict is two or more enabled packs that Load the same target (hard) or edit the same image, map or data
// entry (soft).
type Conflict struct {
	Kind     string   `json:"kind"`
	Target   string   `json:"target"`
	PackIDs  []ModID  `json:"packIds"`
	Names    []string `json:"names"`
	Keys     []string `json:"keys"`
	WinnerID ModID    `json:"winnerId"`
	// WinnerName is the winning pack's own name; empty when WinnerKind names no single winner.
	WinnerName string   `json:"winnerName"`
	WinnerKind string   `json:"winnerKind"`
	Overridden []string `json:"overridden"`
	// Cosmetic marks an edit conflict whose every overlap is harmless: shown, never counted.
	Cosmetic bool `json:"cosmetic"`
	// Fixes are settings that switch off every clashing edit of one pack.
	Fixes    []Fix      `json:"fixes"`
	Evidence []Evidence `json:"evidence"`
	// Note is why a cosmetic conflict cannot hurt play, when one reason covers every overlap.
	Note *Note `json:"note,omitempty"`
	// Farms is, per pack in PackIDs order, the farm types its clashing patches are all limited to: lower-case
	// FarmType values, or a custom farm's Data/AdditionalFarms id (nil for a pack that applies on any farm); nil
	// when no pack is limited.
	Farms [][]string `json:"farms,omitempty"`
	// Cells are the 16-pixel cells, as "x,y", that every pack's clashing patches write; empty when they share none.
	Cells []string `json:"cells,omitempty"`
}

// Note is one reason a conflict is cosmetic. Kind "balance": the packs only set prices and costs, so one mod's
// prices win. Kind "day": the overlap only applies on Value, one day or festival. Kind "farm": it only applies on
// Value farm types, which none of the profile's saves uses.
type Note struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

// Fix sets one on/off field of a pack (Key, ID) to Value, which turns off all of that pack's edits in the
// conflict; Current is the field's value now.
type Fix struct {
	Key     string `json:"key"`
	ID      ModID  `json:"id"`
	Name    string `json:"name"`
	Field   string `json:"field"`
	Current string `json:"current"`
	Value   string `json:"value"`
}

// Hint is a pack setting that would settle a clash. Description is the pack's own schema text; BecausePack names
// the pack whose load beats this one when the current value has no effect.
type Hint struct {
	Key         string   `json:"key"`
	ID          ModID    `json:"id"`
	Name        string   `json:"name"`
	Field       string   `json:"field"`
	Current     string   `json:"current"`
	Suggested   []string `json:"suggested"`
	For         []ModID  `json:"for"`
	ForNames    []string `json:"forNames"`
	Description string   `json:"description"`
	// Variant marks a picker (a palette, recolour or similar choice) whose values the pack maps to mods.
	Variant bool `json:"variant,omitempty"`
	// CurrentFor names the mod the current value is for when that mod is not enabled.
	CurrentFor string `json:"currentFor,omitempty"`
	// BecausePack and DeadTarget mark a setting with no effect: BecausePack loads DeadTarget over the pack's own.
	BecausePack string `json:"becausePack,omitempty"`
	DeadTarget  string `json:"deadTarget,omitempty"`
}

// Evidence is one clashing patch of a pack in a Conflict.
type Evidence struct {
	PackName string `json:"packName"`
	PackID   ModID  `json:"packId"`
	Source   string `json:"source"`
	Index    int    `json:"index"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	ToArea   string `json:"toArea,omitempty"`
	FromArea string `json:"fromArea,omitempty"`
	// When is the patch's condition, for the caller to word.
	When     When   `json:"when"`
	Priority string `json:"priority"`
	FromFile string `json:"fromFile,omitempty"`
	CropX    int    `json:"cropX"`
	CropY    int    `json:"cropY"`
	CropW    int    `json:"cropW"`
	CropH    int    `json:"cropH"`
	// Keys are the data entries and fields this patch sets that another pack's patch also sets.
	Keys []string `json:"keys"`
}

// Shadowed is one enabled pack that later packs overwrite everywhere it writes.
type Shadowed struct {
	Key     string   `json:"key"`
	ID      ModID    `json:"id"`
	Name    string   `json:"name"`
	By      []ModRef `json:"by"`
	Bundled bool     `json:"bundled,omitempty"`
}

// TilesheetUse is a pack that only loads map and tilesheet assets: Enabled when a switched-on mod uses it, and
// Names of the switched-off mods that do.
type TilesheetUse struct {
	Key     string   `json:"key"`
	ID      ModID    `json:"id"`
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled,omitempty"`
	Names   []string `json:"names,omitempty"`
}

// ModRef names one enabled mod.
type ModRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}
