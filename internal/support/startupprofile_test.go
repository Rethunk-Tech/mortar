package support

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	pprofile "github.com/google/pprof/profile"
)

func TestAStartNobodyAskedToProfileRecordsNothing(t *testing.T) {
	t.Setenv("MORTAR_PROFILE_STARTUP", "")
	dir := t.TempDir()
	StartStartupProfile(dir)
	if _, err := os.Stat(filepath.Join(dir, StartupProfileDir)); !os.IsNotExist(err) {
		t.Fatalf("profile folder made without a request: %v", err)
	}
}

func TestTheDiagnosticsCPUProfileNamesProgramFilesWithoutTheirFolders(t *testing.T) {
	var raw bytes.Buffer
	p := &pprofile.Profile{
		SampleType: []*pprofile.ValueType{{Type: "cpu", Unit: "nanoseconds"}},
		Mapping:    []*pprofile.Mapping{{ID: 1, File: "/home/someone/Apps/mortar.appimage"}},
	}
	if err := p.Write(&raw); err != nil {
		t.Fatal(err)
	}
	b, err := cpuProfileWithoutPaths(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	got, err := pprofile.ParseData(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mapping[0].File != "mortar.appimage" {
		t.Fatalf("mapping file = %q", got.Mapping[0].File)
	}
	if _, err := cpuProfileWithoutPaths(nil); err == nil {
		t.Fatal("no profile still produced a file")
	}
}
