package dotnet

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// PluginsIn lists the plugins every DLL under dir declares.
func PluginsIn(dir string) []Plugin {
	var out []Plugin
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".dll") {
			return nil
		}
		plugins, _ := Plugins(path)
		out = append(out, plugins...)
		return nil
	})
	return out
}
