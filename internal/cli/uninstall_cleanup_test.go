package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-gamelocator/steam"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

func TestConfirmAcceptsYesAndRefusesNoOrAnUnreadableAnswer(t *testing.T) {
	for _, answer := range []string{"y", "Y", "yes", "YES"} {
		var errOut bytes.Buffer
		c := &cmd{in: strings.NewReader(answer + "\n"), errOut: &errOut}
		if err := c.confirm("x", "Continue?"); err != nil {
			t.Fatalf("%q: %v", answer, err)
		}
		if !strings.Contains(errOut.String(), "Continue? [y/N] ") {
			t.Errorf("%q: prompt %q", answer, errOut.String())
		}
	}
	for _, answer := range []string{"n", "no", "maybe"} {
		c := &cmd{in: strings.NewReader(answer + "\n"), errOut: &bytes.Buffer{}}
		if err := c.confirm("x", "?"); err == nil || err.Error() != "cancelled" {
			t.Fatalf("%q: %v", answer, err)
		}
	}
	c := &cmd{in: strings.NewReader(""), errOut: &bytes.Buffer{}}
	if err := c.confirm("x", "?"); err == nil || err.Error() == "cancelled" {
		t.Fatalf("an empty stdin read = %v", err)
	}
}

// steamWithShortcuts is a Steam root with one account whose library holds Mortar's two profile shortcuts beside
// entries that must stay.
func steamWithShortcuts(t *testing.T, exe, other string) (root, shortcuts string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "steamapps"), 0o750); err != nil {
		t.Fatal(err)
	}
	write("config/loginusers.vdf", `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`)
	st := steam.Steam{Root: root}
	for _, sc := range []steam.Shortcut{
		{Name: "Farm (Stardew Valley)", Exe: exe, StartDir: filepath.Dir(exe), LaunchOptions: "--play=stardew:Farm", Key: "--play=Farm"},
		{Name: "Mine (Stardew Valley)", Exe: exe, StartDir: filepath.Dir(exe), LaunchOptions: "--play=stardew:Mine", Key: "--play=Mine"},
		{Name: "Not Mortar's", Exe: exe, StartDir: filepath.Dir(exe), LaunchOptions: "%command%", Key: "%command%"},
		{Name: "Something Else", Exe: other, StartDir: filepath.Dir(other), LaunchOptions: "--play=stardew:Else", Key: "--play=Else"},
	} {
		if _, err := st.AddShortcut(sc, steam.EditOptions{BackupSuffix: appversion.SteamBackupSuffix}); err != nil {
			t.Fatal(err)
		}
	}
	return root, filepath.Join(root, "userdata", "39734274", "config", "shortcuts.vdf")
}

func TestRemoveSteamShortcutsDropsOnlyMortarsAndKeepsABackup(t *testing.T) {
	dir := t.TempDir()
	exe, other := filepath.Join(dir, "mortar"), filepath.Join(dir, "else")
	for _, p := range []string{exe, other} {
		if err := os.WriteFile(p, []byte("MZ"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, shortcuts := steamWithShortcuts(t, exe, other)
	// No backup is left over from building the library, so the one the removal takes is the one under test.
	if err := os.Remove(shortcuts + appversion.SteamBackupSuffix); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(shortcuts)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	c := &cmd{out: &out}
	if err := c.removeSteamShortcuts(t.TempDir(), exe, []string{root}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "Removed 2 Mortar shortcuts from Steam\n" {
		t.Fatalf("printed %q", got)
	}
	backup, err := os.ReadFile(shortcuts + appversion.SteamBackupSuffix)
	if err != nil || !bytes.Equal(backup, before) {
		t.Fatalf("the backup holds %q, %v; want the file as it was", backup, err)
	}
	body, err := os.ReadFile(shortcuts)
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"--play=stardew:Farm", "--play=stardew:Mine"} {
		if bytes.Contains(body, []byte(gone)) {
			t.Errorf("%s survived: %q", gone, body)
		}
	}
	for _, kept := range []string{"Not Mortar's", "%command%", "--play=stardew:Else"} {
		if !bytes.Contains(body, []byte(kept)) {
			t.Errorf("%q was removed: %q", kept, body)
		}
	}

	out.Reset()
	if err := c.removeSteamShortcuts(t.TempDir(), exe, []string{root}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Errorf("a second run printed %q", out.String())
	}
}

func TestBackupsRestoreAsksBeforeItRestores(t *testing.T) {
	for _, tc := range []struct {
		answer string
		calls  int
		err    string
	}{
		{"n\n", 0, "cancelled"},
		{"yes\n", 1, ""},
	} {
		var out, errOut bytes.Buffer
		var calls []call
		c := &cmd{
			in: strings.NewReader(tc.answer), out: &out, errOut: &errOut,
			call: fake(map[string]any{"backups.restore": nil}, &calls),
			args: []string{"backups", "restore", "auto-1", "Farm_1"}, game: "stardew",
		}
		err := c.backupsRestore("stardew")
		if tc.err != "" {
			if err == nil || err.Error() != tc.err {
				t.Fatalf("%q: %v", tc.answer, err)
			}
		} else if err != nil {
			t.Fatalf("%q: %v", tc.answer, err)
		}
		if !strings.Contains(errOut.String(), "[y/N]") {
			t.Errorf("%q: never asked: %q", tc.answer, errOut.String())
		}
		if len(calls) != tc.calls {
			t.Fatalf("%q: calls %v", tc.answer, calls)
		}
		if tc.calls == 1 {
			p := calls[0].params
			if calls[0].method != "backups.restore" || p.Name != "auto-1" || p.Game != "stardew" || len(p.IDs) != 1 || p.IDs[0] != "Farm_1" {
				t.Fatalf("params %+v", p)
			}
			if !strings.Contains(out.String(), "Restored auto-1.") {
				t.Fatalf("printed %q", out.String())
			}
		}
	}
}
