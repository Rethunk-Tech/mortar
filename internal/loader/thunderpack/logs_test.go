package thunderpack

import (
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// The lines are the loaders' own format strings: ReturnOfModdingBase logger.cpp and lua_module.cpp, GDWeave's ModLoader
// with Serilog's file template, lovely-core's log.rs, module.rs and pattern.rs, and Northstar's modmanager.cpp with its
// nslog pattern.
func TestAnalyzersReadEachLoadersFailureLines(t *testing.T) {
	type f = loader.Finding
	for name, tc := range map[string]struct {
		analyze func(string) []loader.Finding
		log     string
		want    []f
	}{
		"ReturnOfModding": {
			analyzeROM, "[12:3:44.0123456][INFO/main.cpp:80] ReturnOfModding\n" +
				"[12:3:45.0000001][ERROR/lua_module.cpp:156] Owner-BadMod-1.2.0 failed to load: attempt to index a nil value\n",
			[]f{{Kind: "load-exception", Plugin: "Owner-BadMod-1.2.0", Message: "attempt to index a nil value", Line: 2, Source: "LogOutput.log"}},
		},
		"GDWeave": {
			analyzeGDWeave, "2024-05-01 12:00:00.123 +00:00 [INF] This is GDWeave 2.0.0\n" +
				"2024-05-01 12:00:01.123 +00:00 [WRN] Mod Owner.Pack has missing/invalid dependencies: \"Owner.Lib\", \"Other.Lib\"\n" +
				"2024-05-01 12:00:01.200 +00:00 [WRN] Failed to load mod at C:\\p\\GDWeave\\mods\\Bad-Mod\n",
			[]f{
				{Kind: "missing-dependency", Plugin: "Owner.Pack", Message: "missing or invalid dependencies: \"Owner.Lib\", \"Other.Lib\"", Dependency: "Owner.Lib", Line: 2, Source: "GDWeave.log"},
				{Kind: "load-exception", Plugin: "Bad-Mod", Message: "failed to load", Line: 3, Source: "GDWeave.log"},
			},
		},
		"Lovely": {
			analyzeLovely, "INFO - [♥] Lovely 0.8.0\n" +
				"WARN - [♥] Pattern 'x = 1' on target 'functions/misc_functions.lua' for pattern patch from /p/mods/Steamodded/lovely/core.toml resulted in no matches\n" +
				"ERROR - [♥] Failed to load module foo for module patch from /p/mods/Bad/lovely/m.toml:\n" +
				"ERROR - [♥] Error: [string \"foo\"]:3: unexpected symbol\n",
			[]f{
				{Kind: "patch-exception", Plugin: "Steamodded", Message: "a patch for functions/misc_functions.lua matched nothing in the game's code", Line: 2, Source: "lovely.log", Code: "patchMatchedNothing", Params: map[string]string{"name": "functions/misc_functions.lua"}},
				{Kind: "load-exception", Plugin: "Bad", Message: "module foo failed: [string \"foo\"]:3: unexpected symbol", Line: 3, Source: "lovely.log", Code: "moduleFailedWith", Params: map[string]string{"name": "foo", "error": "[string \"foo\"]:3: unexpected symbol"}},
			},
		},
		"Northstar": {
			analyzeNorthstar, "[2024-05-01] [12:00:00] [NORTHSTAR] [info] 'Good' loaded successfully, version 1.0.0\n" +
				"[2024-05-01] [12:00:00] [NORTHSTAR] [warning] Mod file at '/p/R2Northstar/mods/Author-Broken-1.0.0/mod.json' failed to load\n",
			[]f{{Kind: "load-exception", Plugin: "Author-Broken-1.0.0", Message: "its mod.json failed to load", Line: 2, Source: "nslog.txt"}},
		},
	} {
		if got := tc.analyze(tc.log); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", name, got, tc.want)
		}
	}
}
