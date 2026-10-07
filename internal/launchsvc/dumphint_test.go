package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestFaultsAreClassifiedByWhoseCodeTheyAre(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	if err := os.MkdirAll(filepath.Join(folder, "plugins"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "plugins", "Native.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	mods := []profile.Installed{
		{Key: "a", Enabled: true, EntryDll: "CoolMod.dll"},
		{Key: "b", Enabled: true, Folder: folder},
		{Key: "c", Enabled: false, EntryDll: "Off.dll"},
	}
	exes := []string{"Lethal Company.exe"}
	for module, want := range map[string]string{
		"coolmod.dll": "a", "Native.dll": "b", "Off.dll": "",
	} {
		kind, m := classifyFault(module, exes, mods)
		if m.Key != want || (want != "" && kind != faultMod) || (want == "" && kind == faultMod) {
			t.Errorf("%s: kind %d mod %q, want %q", module, kind, m.Key, want)
		}
	}
	for module, want := range map[string]faultKind{
		"Lethal Company.exe": faultGame, "UnityPlayer.dll": faultUnity, "UnityPlayer": faultUnity,
		"mono-2.0-bdwgc.dll": faultMono, "libmonosgen-2.0.so.1": faultMono, "random.dll": faultUnknown, "": faultUnknown,
	} {
		if kind, _ := classifyFault(module, exes, mods); kind != want {
			t.Errorf("%s: kind %d, want %d", module, kind, want)
		}
	}
}
