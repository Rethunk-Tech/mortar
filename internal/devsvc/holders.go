package devsvc

import (
	"io/fs"
	"path/filepath"
	"slices"
)

// maxHolderPaths bounds how much of a project is asked about when it cannot be moved: its folders first, since a
// terminal or a file watcher holds a folder, then its files.
const maxHolderPaths = 4000

// projectPaths lists dir's folders, then its files, up to maxHolderPaths; a .git folder's contents are left out.
func projectPaths(dir string) []string {
	var dirs, files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		switch {
		case len(dirs)+len(files) >= maxHolderPaths:
			return fs.SkipAll
		case err != nil:
			return fs.SkipDir
		case d.IsDir():
			dirs = append(dirs, path)
			if d.Name() == ".git" {
				return fs.SkipDir
			}
		default:
			files = append(files, path)
		}
		return nil
	})
	return slices.Concat(dirs, files)
}
