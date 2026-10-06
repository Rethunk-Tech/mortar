// Package framework runs the analyzers of framework mods: a framework mod in the profile (Content Patcher) turns on
// the checks that read the content packs written for it.
package framework

import (
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Mod is one mod of the profile.
type Mod struct {
	Key string
	// SourceKind is the entry's source: "local" for an archive, "smapi" for the loader's own mods, "mortar" for the console bridge, "nexus" for a download.
	SourceKind string
	// SourceVersion is the version of the download the entry came from (a Nexus file's version). Authors
	// often leave some manifests of a multi-part download unbumped, so it can be newer than Version.
	SourceVersion string
	// SourceModID is the entry's Nexus mod id, 0 for another source, and SourceCategory the Nexus file's category
	// when installed (MAIN, OPTIONAL, ...).
	SourceModID    int
	SourceCategory string
	// SourceName and SourceRepo are the entry's source name (a Thunderstore "Namespace-Name") and GitHub repo.
	SourceName string
	SourceRepo string
	// SourceDigest is the downloaded file's "sha512:<hex>" where its site published one.
	SourceDigest  string
	Enabled       bool
	Pinned        bool
	SkipVersion   string
	SkipSources   []string
	IgnoreUpdates bool
	UpdateChannel string
	// LoadAfter is mod ids this pack should load after when the user made it win an edit conflict.
	LoadAfter []mod.ID
	// Folder is the mod's directory in the profile, used to read Content Patcher content.json.
	Folder string
	manifest.Manifest
}

// AssetConflict is two or more enabled Content Patcher packs that Load the same target (hard)
// or EditImage/EditMap the same target, or EditData the same entry or field (soft).
type AssetConflict struct {
	Kind       string   `json:"kind"` // "load" (hard) or "edit" (soft)
	Target     string   `json:"target"`
	PackIDs    []mod.ID `json:"packIds"`
	Names      []string `json:"names"`
	Keys       []string `json:"keys"`
	WinnerID   mod.ID   `json:"winnerId"`
	WinnerName string   `json:"winnerName"`
	Overridden []string `json:"overridden"`
	// Cosmetic marks an edit conflict whose every overlap is harmless (see harmless): shown, never counted.
	Cosmetic bool `json:"cosmetic"`
	// Fixes are settings that switch off every clashing edit of one pack.
	Fixes    []ConflictFix      `json:"fixes"`
	Info     string             `json:"info,omitempty"`
	Evidence []ConflictEvidence `json:"evidence"`
}

// ConflictEvidence is one clashing patch of a pack in an AssetConflict.
type ConflictEvidence struct {
	PackName string `json:"packName"`
	PackID   mod.ID `json:"packId"`
	Source   string `json:"source"`
	Index    int    `json:"index"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	ToArea   string `json:"toArea,omitempty"`
	FromArea string `json:"fromArea,omitempty"`
	When     string `json:"when,omitempty"`
	Priority string `json:"priority"`
	FromFile string `json:"fromFile,omitempty"`
	CropX    int    `json:"cropX"`
	CropY    int    `json:"cropY"`
	CropW    int    `json:"cropW"`
	CropH    int    `json:"cropH"`
	// Keys are the data entries or fields this patch sets that another pack's patch also sets.
	Keys []string `json:"keys"`
}

// ConflictFix sets one on/off field of a pack (Key, id) to Value, which turns off all of that pack's edits
// in the conflict; Current is the field's value now.
type ConflictFix struct {
	Key     string `json:"key"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Field   string `json:"field"`
	Current string `json:"current"`
	Value   string `json:"value"`
}

// SettingHint is a Content Patcher compatibility setting that is not enabled for the profile.
type SettingHint struct {
	Key         string   `json:"key"`
	ID          mod.ID   `json:"id"`
	Name        string   `json:"name"`
	Field       string   `json:"field"`
	Current     string   `json:"current"`
	Suggested   []string `json:"suggested"`
	For         []mod.ID `json:"for"`
	ForNames    []string `json:"forNames"`
	Description string   `json:"description"`
	// Variant marks a picker (a palette, recolour or similar choice) whose values the pack maps to mods.
	// CurrentFor names the mod the current value is for when that mod is not enabled.
	Variant    bool   `json:"variant"`
	CurrentFor string `json:"currentFor"`
}

// Cleanup is a framework mod that no enabled mod currently needs.
type Cleanup struct {
	Key    string `json:"key"`
	ID     mod.ID `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

// Redundant is an enabled mod that adds nothing beside the others: "superseded" when its replacement is enabled too,
// "shadowed" when later packs overwrite every edit it makes, "sameJob" when another enabled C# mod changes the same game
// members (Covered when a larger mod changes everything this one does).
type Redundant struct {
	Kind    string   `json:"kind"`
	Key     string   `json:"key"`
	ID      mod.ID   `json:"id"`
	Name    string   `json:"name"`
	By      []ModRef `json:"by"`
	Detail  string   `json:"detail,omitempty"`
	Covered bool     `json:"covered,omitempty"`
}

// ModRef names one enabled mod.
type ModRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}
