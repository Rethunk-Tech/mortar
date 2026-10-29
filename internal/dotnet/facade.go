// Package dotnet reads which BepInEx plugins and Harmony patches a compiled .NET assembly declares, through
// go-dotnet, and orders plugins the way BepInEx loads them.
package dotnet

import godotnet "github.com/Rethunk-Tech/go-dotnet"

type (
	Plugin   = godotnet.Plugin
	Relation = godotnet.Relation
	Declared = godotnet.Declared
	Patch    = godotnet.Patch
	Member   = godotnet.Member
	Melon    = godotnet.Melon
)

const (
	HardDependency  = godotnet.HardDependency
	SoftDependency  = godotnet.SoftDependency
	Incompatible    = godotnet.Incompatible
	PatchSkip       = godotnet.PatchSkip
	PatchPrefix     = godotnet.PatchPrefix
	PatchPostfix    = godotnet.PatchPostfix
	PatchTranspiler = godotnet.PatchTranspiler
	PatchFinalizer  = godotnet.PatchFinalizer
	PatchHook       = godotnet.PatchHook
	SymbolsPortable = godotnet.SymbolsPortable
	SymbolsWindows  = godotnet.SymbolsWindows
)

var (
	Writes          = godotnet.Writes
	Scan            = godotnet.Scan
	Patches         = godotnet.Patches
	PatchKindByName = godotnet.PatchKindByName
	Members         = godotnet.Members
	Usages          = godotnet.Usages
	Identity        = godotnet.Identity
	SourceLine      = godotnet.SourceLine
	ScanMelon       = godotnet.ScanMelon
	MelonOrder      = godotnet.MelonOrder
)
