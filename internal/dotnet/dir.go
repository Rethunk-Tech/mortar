package dotnet

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PluginsIn lists the plugins every DLL under dir declares.
func PluginsIn(dir string) []Plugin {
	var out []Plugin
	for _, path := range dllsIn(dir) {
		plugins, _ := Plugins(path)
		out = append(out, plugins...)
	}
	return out
}

// dllsIn is every DLL under dir, which must be absolute. The walk stays inside dir and keeps only regular files, so a
// symlinked DLL or folder that points elsewhere is never read.
func dllsIn(dir string) []string {
	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		return nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil
	}
	defer func() { _ = root.Close() }()
	var out []string
	_ = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if d.Type().IsRegular() && strings.EqualFold(filepath.Ext(rel), ".dll") {
			out = append(out, filepath.Join(dir, filepath.FromSlash(rel)))
		}
		return nil
	})
	return out
}
