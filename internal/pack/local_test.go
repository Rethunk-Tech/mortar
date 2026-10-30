package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestStageNamesTheRecordedVersionAndAuthor(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := LocalMod{Name: "QALocal", Author: "QA", Version: "0.1.0", Files: []File{{Path: "QALocal/QALocal.dll", Data: []byte("dll")}}}
	if err := m.Stage(dir); err != nil {
		t.Fatal(err)
	}
	if !safefs.IsFile(filepath.Join(dir, "BepInEx", "plugins", "QALocal", "QALocal.dll")) {
		t.Fatal("the plugin was not staged")
	}
	b, err := safefs.ReadFile(filepath.Join(dir, "manifest.json"))
	var got map[string]string
	if err != nil || json.Unmarshal(b, &got) != nil || got["name"] != "QALocal" || got["version_number"] != "0.1.0" || got["author"] != "QA" {
		t.Fatalf("manifest %s (%v)", b, err)
	}
}

func TestStageWritesNoManifestWithNothingToRecord(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := (LocalMod{Name: "Bare", Files: []File{{Path: "Bare/Bare.dll", Data: []byte("dll")}}}).Stage(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("a manifest was written for a mod with no version or author (%v)", err)
	}
}
