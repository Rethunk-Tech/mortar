package installer

import (
	"slices"
	"strings"
)

// Core mods (CET, redscript, RED4ext and the rest) are installed as packed; the pipeline only has to recognise them
// ahead of the mod installers so their files are not mistaken for a mod's (Vortex's installer.core*.ts).

const (
	audiowareCore   = "red4ext/plugins/audioware/audioware.dll"
	tweakXLCore     = "red4ext/plugins/TweakXL/TweakXL.dll"
	archiveXLCore   = "red4ext/plugins/ArchiveXL/ArchiveXL.dll"
	inputLoaderCore = "red4ext/plugins/input_loader/input_loader.dll"
	modSettingsCore = "red4ext/plugins/mod_settings/mod_settings.dll"
	ammCore         = ammBase + "/init.lua"
	cetCore         = "bin/x64/plugins/cyber_engine_tweaks.asi"
	cyberscriptASI  = "bin/x64/plugins/ImmersiveRoleplayFramework.asi"
	cyberscriptCET  = cetPrefix + "/quest_mod"
	cyberscriptPlug = cetPrefix + "/CPStyling"
)

var (
	redscriptCore           = []string{"engine/config/base/scripts.ini", "engine/tools/scc.exe", "r6/config/cybercmd/scc.toml"}
	redscriptCoreDeprecated = []string{"engine/config/base/scripts.ini", "engine/tools/scc.exe", "r6/scripts/redscript.toml"}
	red4extOneNine          = []string{"bin/x64/winmm.dll", "red4ext/LICENSE.txt", "red4ext/THIRD_PARTY_LICENSES.txt", "red4ext/RED4ext.dll"}
	red4extOneSeven         = []string{"bin/x64/d3d11.dll", "red4ext/LICENSE.txt", "red4ext/THIRD_PARTY_LICENSES.txt", "red4ext/RED4ext.dll"}
	red4extDeprecated       = []string{"bin/x64/powrprof.dll", "red4ext/LICENSE.txt", "red4ext/RED4ext.dll"}
	cybercatRequired        = []string{
		"CP2077SaveEditor.exe", "D3DCompiler_47_cor3.dll", "e_sqlite3.dll", "kraken.dll", "PenImc_cor3.dll",
		"PresentationNative_cor3.dll", "vcruntime140_cor3.dll", "wpfgfx_cor3.dll",
	}
)

func (c *cpCtx) hasAll(paths []string) bool {
	return !slices.ContainsFunc(paths, func(p string) bool { return !c.t.pathInTree(p) })
}

func coreCET(c *cpCtx) (outcome, bool, error) {
	if !c.t.pathInTree(cetCore) {
		return outcome{}, false, nil
	}
	return outcome{kind: "core-cet", moves: same(c.t.files)}, true, nil
}

func coreRedscript(c *cpCtx) (outcome, bool, error) {
	switch {
	case c.hasAll(redscriptCore):
		return outcome{kind: "core-redscript", moves: same(c.t.files)}, true, nil
	case c.hasAll(redscriptCoreDeprecated):
		c.note(NoteDeprecatedCore, "this is an old version of redscript; update it from the mod's page")
		return outcome{kind: "core-redscript-deprecated", moves: same(c.t.files)}, true, nil
	}
	return outcome{}, false, nil
}

func coreRed4ext(c *cpCtx) (outcome, bool, error) {
	switch {
	case c.hasAll(red4extOneNine), c.hasAll(red4extOneSeven):
		return outcome{kind: "core-red4ext", moves: same(c.t.files)}, true, nil
	case c.hasAll(red4extDeprecated):
		c.note(NoteDeprecatedCore, "this is an old version of RED4ext; update it from the mod's page")
		return outcome{kind: "core-red4ext-deprecated", moves: same(c.t.files)}, true, nil
	}
	return outcome{}, false, nil
}

// coreDeprecatedTools refuses the two modding tools Vortex refuses (installer.special.deprecated.ts).
func coreDeprecatedTools(c *cpCtx) (outcome, bool, error) {
	if c.t.pathInTree("csvmerge/CSVMerge.cmd") {
		return outcome{}, false, refuse("CSVMerge has been deprecated.")
	}
	if slices.ContainsFunc(c.t.files, func(f string) bool { return strings.HasPrefix(lower(f), "wolvenkit desktop") }) {
		return outcome{}, false, refuse("WolvenKit Desktop cannot be installed as a mod. Install it on its own.")
	}
	if c.t.pathInTree("WolvenKit CLI/WolvenKit.CLI.exe") {
		return outcome{}, false, refuse("WolvenKit installation has been deprecated.")
	}
	return outcome{}, false, nil
}

// coreFile recognises a core mod by the one file every release of it ships.
func coreFile(identifier, kind string) func(*cpCtx) (outcome, bool, error) {
	return func(c *cpCtx) (outcome, bool, error) {
		if !c.t.pathInTree(identifier) {
			return outcome{}, false, nil
		}
		return outcome{kind: "core-" + kind, moves: same(c.t.files)}, true, nil
	}
}

// coreCyberCAT puts the save editor's files in CyberCAT/ below the game folder, and refuses an archive missing any.
func coreCyberCAT(c *cpCtx) (outcome, bool, error) {
	if !slices.ContainsFunc(cybercatRequired, c.t.pathInTree) {
		return outcome{}, false, nil
	}
	if !c.hasAll(cybercatRequired) {
		return outcome{}, false, refuse("CyberCAT archive seems to be missing required files!")
	}
	return outcome{kind: "core-cybercat", moves: moveAll(c.t.files, "", cybercatBase)}, true, nil
}

// coreCyberScript is CyberScript's ASI and CET files and nothing else.
func coreCyberScript(c *cpCtx) (outcome, bool, error) {
	if !c.t.pathInTree(cyberscriptCET + "/init.lua") {
		return outcome{}, false, nil
	}
	var files []string
	if c.t.pathInTree(cyberscriptASI) {
		files = append(files, c.pathAs(cyberscriptASI))
	}
	files = append(files, c.t.filesUnder(cyberscriptCET, nil)...)
	files = append(files, c.t.filesUnder(cyberscriptPlug, nil)...)
	if len(files) != len(c.t.files) {
		return outcome{}, false, refuse("Found files outside known CyberScript directories!")
	}
	return outcome{kind: "core-cyberscript", moves: same(files)}, true, nil
}
