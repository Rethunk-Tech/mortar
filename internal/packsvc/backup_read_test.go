package packsvc

import (
	"archive/zip"
	"os"
	"path/filepath"
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
		"saves/a:b.txt":                 "colon",
		"saves/Dir/x":                   "upper",
		"saves/dir/x":                   "lower",
		"changed/Mods/con .cfg":         "con",
		"changed/Mods/x?.json":          "q",
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
