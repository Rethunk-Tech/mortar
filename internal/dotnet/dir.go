package dotnet

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// PluginsIn lists the plugins every DLL under dir declares.
func PluginsIn(dir string) []Plugin { return ScanDir(dir).Plugins }

// ScanDir reads what every DLL under dir declares about its plugins.
func ScanDir(dir string) Declared {
	var out Declared
	for _, d := range ScanEach(dir) {
		out.Plugins = append(out.Plugins, d.Plugins...)
		out.Relations = append(out.Relations, d.Relations...)
	}
	return out
}

// ScanEach reads what each DLL under dir declares, one Declared per DLL.
func ScanEach(dir string) []Declared {
	paths := DLLs(dir)
	out := make([]Declared, 0, len(paths))
	for _, path := range paths {
		d, _ := Scan(path)
		out = append(out, d)
	}
	return out
}

// DLLs lists every DLL under dir, which must be absolute. The walk stays inside dir and keeps only regular files, so a
// symlinked DLL or folder that points elsewhere is never read.
func DLLs(dir string) []string {
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

// Owners maps every name a BepInEx log gives a plugin, lower-cased, to the index in dirs of the folder whose DLLs
// declare it: its GUID, its name and "name version". An exception's stack names a plugin only by its code's root
// namespace, so that is a name too when exactly one folder's plugins use it.
func Owners(dirs []string) map[string]int {
	owners := map[string]int{}
	byNamespace := map[string][]int{}
	for i, dir := range dirs {
		for _, pl := range PluginsIn(dir) {
			for _, name := range []string{pl.GUID, pl.Name, pl.Name + " " + pl.Version} {
				owners[strings.ToLower(name)] = i
			}
			if root, _, _ := strings.Cut(pl.Namespace, "."); root != "" {
				ns := strings.ToLower(root)
				if !slices.Contains(byNamespace[ns], i) {
					byNamespace[ns] = append(byNamespace[ns], i)
				}
			}
		}
	}
	for ns, is := range byNamespace {
		if _, named := owners[ns]; !named && len(is) == 1 {
			owners[ns] = is[0]
		}
	}
	return owners
}
