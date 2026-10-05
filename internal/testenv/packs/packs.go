// Package packs builds Content Patcher packs on disk for tests of the checks that read them.
package packs

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// Manifest is the manifest of a Content Patcher pack.
func Manifest(id string) string {
	return `{"Name":"` + id + `","UniqueID":"` + id + `","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
}

// Disk writes files into a fresh folder and returns the mod installed from it, keyed and named by id.
func Disk(t *testing.T, id string, files map[string]string) framework.Mod {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		testfs.WriteFile(t, root, rel, body)
	}
	im := framework.Mod{Key: id, Enabled: true, Folder: root}
	im.Name, im.UniqueID = id, id
	return FromDisk(im)
}

// FromDisk fills in what the profile scan reads from a mod's manifest and the checks rely on.
func FromDisk(im framework.Mod) framework.Mod {
	raw, err := fsx.ReadFile(filepath.Join(im.Folder, manifest.FileName))
	if err != nil {
		return im
	}
	if m, err := manifest.Parse(raw); err == nil {
		im.ContentPackFor = m.ContentPackFor
	}
	return im
}
