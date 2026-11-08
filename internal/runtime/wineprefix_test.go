package runtime

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func bottlesInstall(prefix string) Install {
	return Install{Platform: "windows", Store: storeBottles, Prefix: prefix, Dir: prefix, GOOS: "linux"}
}

func TestCommandUsesBottlesRunArgv(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "native-bottle")
	if err := os.Mkdir(native, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "bottle.yml"), []byte("Name: \"Display\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Command(bottlesInstall(native), `C:\setup.exe`, "--silent")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bottles-cli", "run", "-b", "Display", "-e", `C:\setup.exe`, "--", "--silent"}
	if !slices.Equal(got, want) {
		t.Fatalf("native Command = %#v, want %#v", got, want)
	}

	flatpak := filepath.Join(dir, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles", "fp-bottle")
	if err := os.MkdirAll(flatpak, 0o750); err != nil {
		t.Fatal(err)
	}
	got, err = Command(bottlesInstall(flatpak), `C:\game.exe`)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"flatpak", "run", "--command=bottles-cli", "com.usebottles.bottles", "run", "-b", "fp-bottle", "-e", `C:\game.exe`}
	if !slices.Equal(got, want) {
		t.Fatalf("flatpak Command = %#v, want %#v", got, want)
	}
}

func TestPrefixTokensUserDir(t *testing.T) {
	dir := t.TempDir()
	bottle := filepath.Join(dir, "bottle")
	users := filepath.Join(bottle, "drive_c", "users")
	if err := os.MkdirAll(filepath.Join(users, "Public"), 0o750); err != nil {
		t.Fatal(err)
	}
	alice := filepath.Join(users, "alice")
	if err := os.Mkdir(alice, 0o750); err != nil {
		t.Fatal(err)
	}
	tok := prefixTokens(bottlesInstall(bottle))
	if tok["home"] != alice {
		t.Fatalf("home = %q, want %q", tok["home"], alice)
	}
}
