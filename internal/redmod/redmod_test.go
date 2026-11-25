package redmod

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// install is a game folder with REDmod's files, whose tool is a script that records its arguments and the modlist it
// was given, then prints out and exits with code.
func install(t *testing.T, out string, code int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in tool is a shell script")
	}
	dir := t.TempDir()
	testfs.WriteAt(t, filepath.Join(dir, launcherPath), []byte("x"))
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$0.args\"\ncat \"${6#-modlist=}\" > \"$0.modlist\"\nprintf '%s' '" + out + "'\nexit " + strconv.Itoa(code) + "\n"
	tool := Tool(dir)
	testfs.WriteAt(t, tool, []byte(script))
	if err := os.Chmod(tool, 0o700); err != nil { //nolint:gosec // G302: the stand-in tool must be executable, in the test's own temp folder
		t.Fatal(err)
	}
	return dir
}

func run(ctx context.Context, exe string, args []string) (int, string, error) {
	out, err := exec.CommandContext(ctx, exe, args...).CombinedOutput() //nolint:gosec // G204: the test's own stand-in tool under t.TempDir(), arguments built by Args
	if x, ok := errors.AsType[*exec.ExitError](err); ok {
		return x.ExitCode(), string(out), nil
	}
	return 0, string(out), err
}

func same(p string) string { return p }

func TestDeployWritesTheModListAndRunsTheTool(t *testing.T) {
	dir := install(t, "", 0)
	if err := Deploy(t.Context(), dir, []string{"b", "a"}, same, run); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(Tool(dir) + ".args")
	want := "deploy\n-reportProgress\n-root\n" + dir + "\n-rttiSchemaPath=" + dir + "/tools/redmod/metadata.json\n-modlist=" + dir + "/V2077/modlist.txt\n"
	if string(args) != want {
		t.Fatalf("args = %q, want %q", args, want)
	}
	if list, _ := os.ReadFile(Tool(dir) + ".modlist"); string(list) != "b\r\na" {
		t.Fatalf("modlist = %q", list)
	}
}

func TestDeployClearsGeneratedFilesOnlyWhenTheOrderChanged(t *testing.T) {
	dir := install(t, "", 0)
	generated := filepath.Join(dir, "r6", "cache", "modded", "keep.bin")
	deploy := func(names ...string) {
		t.Helper()
		if err := Deploy(t.Context(), dir, names, same, run); err != nil {
			t.Fatal(err)
		}
	}
	deploy("a", "b")
	testfs.WriteAt(t, generated, []byte("x"))
	deploy("a", "b")
	if _, err := os.Stat(generated); err != nil {
		t.Fatal("an unchanged list cleared the generated files")
	}
	deploy("b", "a")
	if _, err := os.Stat(generated); err == nil {
		t.Fatal("a reordered list kept the generated files")
	}
}

func TestDeployFailureCarriesTheToolsReason(t *testing.T) {
	dir := install(t, "[DEPLOY] working\nMod x has a bad info.json\nCommandlet deploy has failed.\n", 3)
	err := Deploy(t.Context(), dir, []string{"x"}, same, run)
	var failed *FailedError
	if !errors.As(err, &failed) || failed.Reason != "Mod x has a bad info.json" {
		t.Fatalf("err = %v", err)
	}
}

func TestFailureReason(t *testing.T) {
	for _, c := range []struct {
		code   int
		output string
		want   string
	}{
		{0xC000013A, "anything", "REDmod was interrupted"},
		{1, "", "REDmod exited with code 1"},
		{1, "[DEPLOY] stage\r\nCommandlet deploy has failed.", "REDmod exited with code 1"},
		{1, "oops\nCommandlet deploy has failed.\n", "oops"},
	} {
		if got := FailureReason(c.code, c.output); got != c.want {
			t.Errorf("FailureReason(%d, %q) = %q, want %q", c.code, c.output, got, c.want)
		}
	}
}

func TestInstalledNeedsToolAndLauncher(t *testing.T) {
	dir := install(t, "", 0)
	if !Installed(dir) {
		t.Fatal("a full install is not installed")
	}
	if err := os.Remove(filepath.Join(dir, launcherPath)); err != nil || Installed(dir) {
		t.Fatalf("without the launcher: %v, installed %v", err, Installed(dir))
	}
}

func TestFindingWhenREDmodsAreHereButTheToolIsNot(t *testing.T) {
	game := t.TempDir()
	view := loader.ProfileView{Game: "cyberpunk-2077", Dir: t.TempDir(), InstallDir: game, Files: []string{"Mods/mods/M/info.json", "Mods/archive/pc/mod/a.archive"}}
	if f := (extension{}).Findings(view); len(f) != 1 || f[0].Code != CodeMissing {
		t.Fatalf("findings = %+v", f)
	}
	view.Files = []string{"Mods/archive/pc/mod/a.archive", "Mods/other/info.json"}
	if f := (extension{}).Findings(view); len(f) != 0 {
		t.Fatalf("no REDmod in the profile: %+v", f)
	}
	view.Files = []string{"Mods/mods/M/info.json"}
	testfs.WriteAt(t, filepath.Join(game, launcherPath), []byte("x"))
	testfs.WriteAt(t, Tool(game), []byte("x"))
	if f := (extension{}).Findings(view); len(f) != 0 {
		t.Fatalf("installed: %+v", f)
	}
}

func TestFindingsCarryWhatTheInstallerNoted(t *testing.T) {
	profile := t.TempDir()
	notes := []installer.Note{{Code: installer.NoteUnrecognised, Message: "placed as packed", Files: []string{"a.txt"}}}
	if err := installer.RecordNotes(profile, "nexus-1-2", "Some Mod", notes); err != nil {
		t.Fatal(err)
	}
	if err := installer.RecordNotes(profile, "removed", "Gone", notes); err != nil {
		t.Fatal(err)
	}
	view := loader.ProfileView{Game: "cyberpunk-2077", Dir: profile, EntryKeys: []string{"nexus-1-2"}}
	f := (extension{}).Findings(view)
	if len(f) != 1 || f[0].Plugin != "Some Mod" || f[0].Code != installer.NoteUnrecognised || f[0].Params["count"] != "1" {
		t.Fatalf("findings = %+v", f)
	}
	if err := installer.RecordNotes(profile, "nexus-1-2", "Some Mod", nil); err != nil {
		t.Fatal(err)
	}
	if f := (extension{}).Findings(view); len(f) != 0 {
		t.Fatalf("after clearing: %+v", f)
	}
}

func TestArchiveModListIsOnlyWrittenWhenTheNameOrderDiffers(t *testing.T) {
	game := t.TempDir()
	path := filepath.Join(game, filepath.FromSlash(ArchiveModListPath))
	if err := WriteArchiveModList(game, []string{"b.archive", "a.archive"}); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(path); err != nil || string(b) != "b.archive\r\na.archive" {
		t.Fatalf("modlist = %q, %v", b, err)
	}
	if err := WriteArchiveModList(game, []string{"a.archive", "b.archive"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the default order needs no modlist: %v", err)
	}
}

func TestDeployedIsWhetherAnEarlierDeployLeftREDmods(t *testing.T) {
	dir := install(t, "", 0)
	if Deployed(dir) {
		t.Fatal("a game that never deployed has REDmods")
	}
	if err := Deploy(t.Context(), dir, []string{"a"}, same, run); err != nil || !Deployed(dir) {
		t.Fatalf("after deploying a: %v, deployed %v", err, Deployed(dir))
	}
	if err := Deploy(t.Context(), dir, nil, same, run); err != nil || Deployed(dir) {
		t.Fatalf("after deploying none: %v, deployed %v", err, Deployed(dir))
	}
	if _, err := os.Stat(filepath.Join(dir, "mods")); err != nil {
		t.Fatalf("the tool's mods folder was not made: %v", err)
	}
}

func TestAPlayersOwnArchiveModListIsSetAsideAndGivenBack(t *testing.T) {
	game := t.TempDir()
	path := filepath.Join(game, filepath.FromSlash(ArchiveModListPath))
	testfs.WriteAt(t, path, []byte("mine.archive"))
	if err := WriteArchiveModList(game, []string{"a.archive"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(path); string(b) != "mine.archive" {
		t.Fatalf("a list Mortar does not need replaced the player's: %q", b)
	}
	if err := WriteArchiveModList(game, []string{"b.archive", "a.archive"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(path); string(b) != "b.archive\r\na.archive" {
		t.Fatalf("modlist = %q", b)
	}
	if err := WriteArchiveModList(game, []string{"a.archive", "b.archive"}); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(path); err != nil || string(b) != "mine.archive" {
		t.Fatalf("the player's modlist did not come back: %q, %v", b, err)
	}
}
