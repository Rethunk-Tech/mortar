//go:build !windows

package launch

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

func TestProcesses(t *testing.T) {
	proc := t.TempDir()
	add := func(pid, cmdline string) {
		dir := filepath.Join(proc, pid)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	add("101", "/g/StardewModdingAPI\x00--mods-path\x00/p/a/mods\x00")
	add("102", "/g/StardewModdingAPI\x00--mods-path=/p/b/mods\x00")
	add("103", "/usr/bin/vim\x00StardewModdingAPI\x00")
	add("104", "dotnet\x00/g/StardewModdingAPI.dll\x00--mods-path\x00/p/c/mods\x00")
	add("105", "/g/StardewModdingAPI\x00")
	if err := os.MkdirAll(filepath.Join(proc, "self"), 0o700); err != nil {
		t.Fatal(err)
	}
	procs, err := Processes(proc, "StardewModdingAPI")
	if err != nil || len(procs) != 4 {
		t.Fatalf("procs = %+v, %v", procs, err)
	}
	want := map[string]int{"/p/a/mods": 101, "/p/b/mods": 102, "/p/c/mods": 104}
	for dir, pid := range want {
		var found []int
		for _, p := range procs {
			if p.UsesModsPath(dir) {
				found = append(found, p.PID)
			}
		}
		if !slices.Equal(found, []int{pid}) {
			t.Fatalf("%s matched %v, want [%d]", dir, found, pid)
		}
	}
	if procs[0].UsesModsPath("/p/other/mods") {
		t.Fatal("matched an unrelated mods path")
	}
}

func TestProcessStartTime(t *testing.T) {
	proc := t.TempDir()
	dir := filepath.Join(proc, "7")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// comm holds a space and a parenthesis; starttime is the 22nd field.
	stat := "7 (Stardew (x) y) S 1 7 7 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 12345 0 0\n"
	files := map[string]string{
		"7/cmdline": "/g/StardewModdingAPI\x00", "7/stat": stat, "stat": "cpu 1 2\nbtime 1000000\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(proc, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	procs, err := Processes(proc, "StardewModdingAPI")
	if err != nil || len(procs) != 1 {
		t.Fatalf("procs = %+v, %v", procs, err)
	}
	if want := time.Unix(1000000+123, 0); !procs[0].Start.Equal(want) {
		t.Fatalf("start = %v, want %v", procs[0].Start, want)
	}
}

func TestParseHostProcesses(t *testing.T) {
	out := "12\t1700000000\t/usr/bin/steam\x1f-silent\n34\t1700000100\t/g/Stardew Valley/StardewModdingAPI\x1f--mods-path\x1f/m\x1f\nx\ty\n"
	got := parseHostProcesses(out, "StardewModdingAPI")
	if len(got) != 1 || got[0].PID != 34 || got[0].Start.Unix() != 1700000100 || !got[0].UsesModsPath("/m") {
		t.Fatalf("%+v", got)
	}
}

func TestHostTerminateSignalsThroughHostKill(t *testing.T) {
	var calls []string
	old, oldEnv := sandbox.Output, sandbox.Getenv
	t.Cleanup(func() { sandbox.Output, sandbox.Getenv = old, oldEnv })
	sandbox.Getenv = func(string) string { return "x" }
	alive := true
	sandbox.Output = func(args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if !alive {
			return nil, errors.New("no such process")
		}
		alive = args[len(args)-2] != "-TERM"
		return nil, nil
	}
	if err := Terminate(42, time.Second); err != nil {
		t.Fatal(err)
	}
	want := []string{"--host kill -0 42", "--host kill -TERM 42", "--host kill -0 42"}
	if !slices.Equal(calls, want) {
		t.Fatalf("calls %q", calls)
	}
}
