package datadir

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// CopyTree copies the regular files and folders under src into dst (which may exist), file by file with io.Copy, which
// the OS clones where the filesystem can. Anything else, such as a symlink, is an error.
func CopyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o750)
		case !d.Type().IsRegular():
			return fmt.Errorf("%s is not a regular file", p)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) (err error) {
	in, err := fsx.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := fsx.CreateExcl(dst, 0o600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	_, err = io.Copy(out, in)
	return err
}
