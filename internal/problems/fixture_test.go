package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// cpManifest is the manifest of a Content Patcher pack.
func cpManifest(id string) string {
	return `{"Name":"` + id + `","UniqueID":"` + id + `","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
}

func writeProblemFile(t *testing.T, root, rel, body string) {
	t.Helper()
	testfs.WriteFile(t, root, rel, body)
}

// diskPack writes files into a fresh folder and returns the mod installed from it, keyed and named by id.
func diskPack(t *testing.T, id string, files map[string]string) Installed {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		writeProblemFile(t, root, rel, body)
	}
	return fromDisk(Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
}
