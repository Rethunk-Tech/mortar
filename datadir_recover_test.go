//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// zenity puts a stand-in zenity first on PATH for the test, so findDialogTool finds it as it would the real one.
func zenity(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "zenity"), script)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// pointAtMissing makes the default data folder name a folder that is not there, the way an unplugged drive leaves it.
func pointAtMissing(t *testing.T) (def, missing string) {
	t.Helper()
	testfs.DataHome(t)
	var err error
	if def, err = datadir.DefaultDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(def, 0o700); err != nil {
		t.Fatal(err)
	}
	missing = filepath.Join(t.TempDir(), "unplugged-drive", "mortar")
	if err := datadir.SetLocation(missing); err != nil {
		t.Fatal(err)
	}
	return def, missing
}

// A missing data folder with no dialog to ask through is reported to a command line or the next start's log rather
// than guessed at, and the folder it names is the one the pointer names.
func TestAMissingDataFolderWithNoDialogIsReported(t *testing.T) {
	prev := dialogTools
	dialogTools = nil
	t.Cleanup(func() { dialogTools = prev })
	_, missing := pointAtMissing(t)
	dir, err := dataDirOrRecover()
	if dir != "" || err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("dataDirOrRecover = %q, %v; want the missing folder named", dir, err)
	}
	// The log inside the data folder may be what is missing, so the failure also reaches the default location.
	def, err := datadir.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(def, "startup.log")); !strings.Contains(got, missing) {
		t.Fatalf("startup.log = %q, want the missing folder", got)
	}
}

// Answering "use the default location" drops the relocation pointer and starts on the default folder again.
func TestChoosingTheDefaultLocationRecovers(t *testing.T) {
	zenity(t, "#!/bin/sh\nexit 0\n")
	def, _ := pointAtMissing(t)
	dir, err := dataDirOrRecover()
	if err != nil || dir != def {
		t.Fatalf("dataDirOrRecover = %q, %v; want %q", dir, err, def)
	}
	if _, err := os.Stat(filepath.Join(def, datadir.PointerName)); !os.IsNotExist(err) {
		t.Errorf("the pointer is still there: %v", err)
	}
}

// Answering "choose the data folder" saves the folder picked as the new location, and Dir answers it from then on.
func TestChoosingAFolderRecovers(t *testing.T) {
	picked := filepath.Join(t.TempDir(), "data-on-a-stick")
	testfs.WriteFile(t, picked, "keep", "")
	// The first dialog is the question, answered with its extra button's label on stdout; the second is the folder
	// chooser, which prints the folder it picked.
	count := filepath.Join(t.TempDir(), "calls")
	zenity(t, "#!/bin/sh\nif [ -e "+count+" ]; then printf '%s\\n' "+"'"+filepath.ToSlash(picked)+"'"+"; exit 0; fi\ntouch "+count+"\nprintf '%s\\n' "+"'"+labelChoose()+"'"+"\nexit 1\n")
	_, _ = pointAtMissing(t)
	dir, err := dataDirOrRecover()
	if err != nil || dir != picked {
		t.Fatalf("dataDirOrRecover = %q, %v; want %q", dir, err, picked)
	}
}

// A startup error reaches the dialog tool with the message it holds, offering the log folder only when there is one
// to open, and a tool whose answer names no folder just closes.
func TestShowStartupErrorPassesTheMessageToTheTool(t *testing.T) {
	args := filepath.Join(t.TempDir(), "args")
	zenity(t, "#!/bin/sh\nprintf '%s\\n' \"$*\" > "+args+"\nexit 1\n")
	showStartupError("the drive is unplugged", "")
	got := testfs.Read(t, args)
	if !strings.Contains(got, "the drive is unplugged") || !strings.Contains(got, "--error") {
		t.Fatalf("the dialog got %q", got)
	}
	if strings.Contains(got, labelOpenLog()) {
		t.Errorf("a dialog with no folder offers to open one: %q", got)
	}
	zenity(t, "#!/bin/sh\nprintf '%s\\n' \"$*\" > "+args+"\nexit 1\n")
	showStartupError("the drive is still unplugged", t.TempDir())
	if got := testfs.Read(t, args); !strings.Contains(got, labelOpenLog()) || !strings.Contains(got, "--extra-button") {
		t.Errorf("a startup error with a folder did not offer to open it: %q", got)
	}
}

// writeExecutable writes a script at path and makes it runnable through its folder's root, which the lint accepts
// where a world-readable mode would not.
func writeExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := root.Chmod(filepath.Base(path), 0o700); err != nil {
		t.Fatal(err)
	}
}
