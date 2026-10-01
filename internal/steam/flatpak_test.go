package steam

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHasFilesystem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mortar")
	show := "[Context]\nfilesystems=" + dir + ":ro;xdg-download;\n"
	if !HasFilesystem(show, dir) {
		t.Fatal("granted path not seen")
	}
	if HasFilesystem("[Context]\nfilesystems=xdg-download;\n", dir) {
		t.Fatal("unrelated filesystem")
	}
	if !HasFilesystem("filesystems=home;", dir) {
		t.Fatal("home should grant")
	}
	if HasFilesystem("", dir) {
		t.Fatal("empty show")
	}
}

func TestOverrideCommand(t *testing.T) {
	dir := filepath.Join(string(filepath.Separator), "data", "mortar")
	got := OverrideCommand(dir)
	if !strings.Contains(got, dir+":ro") || !strings.Contains(got, FlatpakID) {
		t.Fatalf("command = %q", got)
	}
}

func TestGrantFilesystemUsesOverrideArgs(t *testing.T) {
	var args []string
	runFlatpak = func(a ...string) ([]byte, error) {
		args = a
		return nil, nil
	}
	orig := runFlatpak
	t.Cleanup(func() { runFlatpak = orig })
	dir := filepath.Join(t.TempDir(), "mortar")
	if err := GrantFilesystem(dir); err != nil {
		t.Fatal(err)
	}
	if len(args) < 4 || args[0] != "override" || args[2] != "--filesystem="+dir+":ro" || args[3] != FlatpakID {
		t.Fatalf("args = %v", args)
	}
	out, err := ShowOverride()
	if err != nil || out != "" {
		t.Fatalf("show = %q, %v", out, err)
	}
}
