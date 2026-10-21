package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/crashdump"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/symsrv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
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

// A native fault is named by the public symbol its offset lies in when the module's PDB is cached (offline here, so
// nothing is fetched), and by module and offset otherwise.
func TestNativeFrameNamesTheFunctionFromACachedPDB(t *testing.T) {
	cv, ok := symsrv.PECodeView(filepath.Join("..", "symsrv", "testdata", "UnityPlayer.dll"))
	if !ok {
		t.Fatal("the fixture has no CodeView record")
	}
	pdb, err := os.ReadFile(filepath.Join("..", "symsrv", "testdata", cv.PDB))
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	testfs.WriteFile(t, cache, cv.PDB+"/"+cv.Key()+"/"+cv.PDB, string(pdb))
	s := &Service{Symbols: &symsrv.Resolver{Dir: cache, Online: func() bool { return false }}}
	for _, c := range []struct {
		fault crashdump.Fault
		want  string
	}{
		{crashdump.Fault{Module: "UnityPlayer.dll", Offset: 0x101c, HasOffset: true, CodeView: cv}, "UnityPlayer!GfxDeviceD3D11::DrawBuffers (+0x1c)"},
		{crashdump.Fault{Module: "UnityPlayer.dll", Offset: 0x101c, HasOffset: true}, "UnityPlayer.dll+0x101c"},
		{crashdump.Fault{Module: "UnityPlayer.dll"}, ""},
	} {
		if got := s.nativeFrame(c.fault); got != c.want {
			t.Errorf("%+v: %q, want %q", c.fault, got, c.want)
		}
	}
}
