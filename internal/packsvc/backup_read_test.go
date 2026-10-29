package packsvc

import (
	"archive/zip"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func writeZip(t *testing.T, entries [][2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "b.zip")
	f, err := safefs.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(e[1]))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}

// A backup writes names straight from disk, so a restore reads names Windows would refuse and case-only twins.
func TestReadBackupReadsEveryNameBackupWrites(t *testing.T) {
	caseInsensitive = false
	disk := map[string]string{
		"store/nexus-1-2/Mod.dll":       "dll",
		"store/nexus-1-2/sub/data.json": "json",
		"saves/Farm_1/Farm_1":           "save",
	}
	// A backup made on Linux holds these, and no Windows file system can: there the restore refuses them.
	if runtime.GOOS != "windows" {
		maps.Copy(disk, map[string]string{
			"saves/a:b.txt":         "colon",
			"saves/Dir/x":           "upper",
			"saves/dir/x":           "lower",
			"changed/Mods/con .cfg": "con",
			"changed/Mods/x?.json":  "q",
		})
	}
	files := map[string]string{"x?.json": "fq", "Cfg/a.cfg": "cfg", "cfg/A.cfg": "cfg2"}
	var entries [][2]string
	entries = append(entries, [2]string{backupDoc, `{"version":1,"game":"stardew","profile":{"name":"Farm"}}`})
	for n, b := range files {
		entries = append(entries, [2]string{backupPrefix + n, b})
	}
	for n, b := range disk {
		entries = append(entries, [2]string{n, b})
	}
	tmp := t.TempDir()
	doc, got, err := readBackup(writeZip(t, entries), tmp)
	if err != nil || doc.Game != "stardew" {
		t.Fatalf("doc %+v, %v", doc, err)
	}
	if len(got) != len(files) {
		t.Fatalf("files = %v", got)
	}
	for n, b := range files {
		if string(got[n]) != b {
			t.Errorf("files/%s = %q", n, got[n])
		}
	}
	for n, b := range disk {
		if data, err := safefs.ReadFile(filepath.Join(tmp, filepath.FromSlash(n))); err != nil || string(data) != b {
			t.Errorf("%s = %q, %v", n, data, err)
		}
	}
}

func TestReadBackupRefusesTraversal(t *testing.T) {
	for _, name := range []string{"saves/../../x", "../x"} {
		path := writeZip(t, [][2]string{{backupDoc, `{"version":1,"game":"g"}`}, {name, "x"}})
		tmp := t.TempDir()
		if _, _, err := readBackup(path, tmp); err == nil {
			t.Errorf("%q was read", name)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(tmp), "x")); err == nil {
			t.Errorf("%q escaped", name)
		}
	}
}

func TestNoteFoldedRefusesCaseTwinsOnlyOnAFoldingHost(t *testing.T) {
	for _, fold := range []bool{true, false} {
		folded := map[string]string{}
		_ = noteFolded(folded, "saves/Farm/Farm", fold)
		err := noteFolded(folded, "saves/farm/farm", fold)
		if fold != (err != nil && strings.Contains(err.Error(), "cannot keep apart")) {
			t.Errorf("fold=%v: err = %v", fold, err)
		}
	}
}

// files/ entries are held in memory, so case twins there restore on every host.
func TestReadBackupKeepsFilesCaseTwinsOnAFoldingHost(t *testing.T) {
	caseInsensitive = true
	t.Cleanup(func() { caseInsensitive = runtime.GOOS == "windows" || runtime.GOOS == "darwin" })
	path := writeZip(t, [][2]string{
		{backupDoc, `{"version":1,"game":"g"}`},
		{backupPrefix + "Config.json", "a"},
		{backupPrefix + "config.json", "b"},
	})
	_, got, err := readBackup(path, t.TempDir())
	if err != nil || string(got["Config.json"]) != "a" || string(got["config.json"]) != "b" {
		t.Fatalf("files = %v, %v", got, err)
	}
}
